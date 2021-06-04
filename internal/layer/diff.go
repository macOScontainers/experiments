package layer

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/macoscontainers/experiments/internal/filesystem"
)

// Provides functionality for generating a filesystem diff by comparing modified files to a base filesystem layer
type DiffGenerator struct {
	
	// The absolute path to the root directory for the base filesystem layer
	BaseDir string
	
	// The absolute path to the root directory of the modified files to be compared against the base filesystem layer
	ModifiedDir string
	
	// The absolute path to the root directory in which to place the generated filesystem diff
	DiffDir string
}

// Recursively computes the diff for a given filesystem subpath compared to the contents of the base filesystem layer
func (diff *DiffGenerator) DiffRecursive(subpath string, subpathDetails fs.DirEntry, dirAdded bool) <-chan error {
	
	// Create a channel to store the result
	result := make(chan error, 1)
	
	// Perform processing in a separate goroutine
	go func() {
		result <- diff.diffRecursiveImp(subpath, subpathDetails, dirAdded)
		close(result)
	}()
	
	return result
}

// The internal implementation of the DiffRecursive() function
func (diff *DiffGenerator) diffRecursiveImp(subpath string, subpathDetails fs.DirEntry, dirAdded bool) error {
	
	// Gather errors from recursive calls to they can be aggregated for the caller
	errorChannels := []<-chan error{}
	
	// DEBUG
	log.Println("Entering subpath", subpath)
	
	// Unless this is the root directory, create the appropriate subdirectory in the diff directory
	if subpath != "" && subpathDetails != nil {
		
		// Create the directory
		dirPath := filepath.Join(diff.DiffDir, subpath)
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return err
		}
		
		// Copy the directory's attributes
		if err := CopyAttributes("", dirPath, subpathDetails); err != nil {
			return err
		}
	}
	
	// List the directory contents for the subpath in the base filesystem layer
	baseEntries, err := filesystem.ReadDirAsMap(filepath.Join(diff.BaseDir, subpath))
	if err != nil {
		return err
	}
	
	// List the directory contents for the subpath in the modified files
	modifiedEntries, err := filesystem.ReadDirAsMap(filepath.Join(diff.ModifiedDir, subpath))
	if err != nil {
		return err
	}
	
	// Identify files and subdirectories that have been modified or removed
	for filename, baseDetails := range baseEntries {
		if !modifiedEntries.Exists(filename) {
			
			// The file or subdirectory has been removed, so generate a whiteout file
			if err := diff.generateWhiteout(subpath, filename); err != nil {
				return err
			}
			
		} else {
			
			// Compare the file or subdirectory to the original version to determine if they differ
			modifiedDetails := modifiedEntries[filename]
			if baseDetails.IsDir() && !modifiedDetails.IsDir() {
				
				// The original entry was a directory that has been removed and replaced with a file of the same name
				
				// Generate a whiteout for the original directory
				if err := diff.generateWhiteout(subpath, filename); err != nil {
					return err
				}
				
				// Mirror the new file to the diff
				if err := diff.mirrorFile(subpath, filename, modifiedDetails); err != nil {
					return err
				}
				
			} else if !baseDetails.IsDir() && modifiedDetails.IsDir() {
				
				// The original entry was a file that has been removed and replaced with a directory of the same name
				
				// Generate a whiteout for the original file
				if err := diff.generateWhiteout(subpath, filename); err != nil {
					return err
				}
				
				// Process the directory recursively
				errorChannels = append(errorChannels, diff.DiffRecursive(filepath.Join(subpath, filename), modifiedDetails, true))
				
			} else if baseDetails.IsDir() && modifiedDetails.IsDir() {
				
				// The original entry was a directory and this has not changed
				
				// TODO: determine whether the directory's attributes have changed
				attributesChanged := false
				
				// Process the directory recursively
				errorChannels = append(errorChannels, diff.DiffRecursive(filepath.Join(subpath, filename), modifiedDetails, attributesChanged))
				
			} else {
				
				// The original entry was a file and this has not changed
				
				// TODO: determine whether the file or its attributes have changed
				//...
				
			}
		}
	}
	
	// Identify files and subdirectories that have been added
	for filename, details := range modifiedEntries {
		if !baseEntries.Exists(filename) {
			
			// Determine whether the entry is a file or a directory
			if details.IsDir() {
				
				// Process the directory recursively
				errorChannels = append(errorChannels, diff.DiffRecursive(filepath.Join(subpath, filename), details, true))
				
			} else {
				
				// Mirror the file to the diff
				if err := diff.mirrorFile(subpath, filename, details); err != nil {
					return err
				}
				
			}
		}
	}
	
	// Aggregate any errors from recursive calls
	var aggregated error
	for _, ch := range errorChannels {
		err := <- ch
		if err != nil {
			aggregated = multierror.Append(err)
		}
	}
	
	// Determine if the directory was an existing directory and should therefore only exist in the diff if it contains differences
	if !dirAdded {
		
		// Retrieve the list of generated differences for the directory that we've just processed
		diffDir := filepath.Join(diff.DiffDir, subpath)
		diffEntries, err := filesystem.ReadDirAsMap(diffDir)
		if err != nil {
			return err
		}
		
		// If there were no differences inside the directory and the directory then remove it from the diff
		if len(diffEntries) == 0 {
			if err := os.RemoveAll(diffDir); err != nil {
				return err
			}
		}
	}
	
	return aggregated
}

// Generates a whiteout file for a file or directory
func (diff *DiffGenerator) generateWhiteout(subpath string, filename string) error {
	
	// Attempt to create a whiteout file
	whiteout, err := os.Create(filepath.Join(diff.DiffDir, subpath, WhiteoutForFile(filename)));
	if err != nil {
		return err
	}
	
	// Close the file if we created it successfully
	whiteout.Close()
	return nil
}

// Mirrors an individual file from the modified files to the diff directory
func (diff *DiffGenerator) mirrorFile(subpath string, filename string, details fs.DirEntry) error {
	return MirrorFileWithAttributes(
		filepath.Join(diff.ModifiedDir, subpath, filename),
		filepath.Join(diff.DiffDir, subpath, filename),
		details,
	)
}
