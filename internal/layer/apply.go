package layer

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/macoscontainers/experiments/internal/filesystem"
)

// Provides functionality for applying a filesystem diff against a base filesystem layer
type DiffApplier struct {
	
	// The absolute path to the root directory for the base filesystem layer
	BaseDir string
	
	// The absolute path to the root directory for the diff to be applied against the base filesystem layer
	DiffDir string
	
	// The absolute path to the root directory in which to place the merged output
	MergedDir string
}

// Recursively applies the diff for a given filesystem subpath to the contents of the base filesystem layer
func (apply *DiffApplier) ApplyRecursive(subpath string, subpathDetails fs.DirEntry, whiteoutInParent bool) <-chan error {
	
	// Create a channel to store the result
	result := make(chan error, 1)
	
	// Perform processing in a separate goroutine
	go func() {
		result <- apply.applyRecursiveImp(subpath, subpathDetails, whiteoutInParent)
		close(result)
	}()
	
	return result
}

// The internal implementation of the ApplyRecursive() function
func (apply *DiffApplier) applyRecursiveImp(subpath string, subpathDetails fs.DirEntry, whiteoutInParent bool) error {
	
	// Gather errors from recursive calls to they can be aggregated for the caller
	errorChannels := []<-chan error{}
	
	// DEBUG
	log.Println("Entering subpath", subpath)
	
	// Unless this is the root directory, create the appropriate subdirectory in the merged output directory
	if subpath != "" && subpathDetails != nil {
		
		// Create the directory
		dirPath := filepath.Join(apply.MergedDir, subpath)
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return err
		}
		
		// Copy the directory's attributes
		if err := CopyAttributes("", dirPath, subpathDetails); err != nil {
			return err
		}
	}
	
	// List the directory contents for the subpath in the diff
	diffEntries, err := filesystem.ReadDirAsMap(filepath.Join(apply.DiffDir, subpath))
	if err != nil {
		return err
	}
	
	// Determine whether the contents of the subpath from the base filesystem layer have been erased by a whiteout file or
	// opaque whiteout file in the parent directory of the diff or an opaque whiteout file the current directory of the diff
	haveOpaqueWhiteout := diffEntries.Exists(OPAQUE_WHITEOUT_FILENAME)
	ignoreBase := whiteoutInParent || haveOpaqueWhiteout
	
	// Only list the directory contents for the subpath in the base filesystem layer if it has not been erased
	baseEntries := make(filesystem.DirEntryMap)
	if !ignoreBase {
		var err error
		baseEntries, err = filesystem.ReadDirAsMap(filepath.Join(apply.BaseDir, subpath))
		if err != nil {
			return err
		}
	}
	
	// Merge the contents of the base filesystem layer into the output directory
	for filename, details := range baseEntries {
		
		// Ignore the file or directory if it has been erased or overwritten
		if !diffEntries.Exists(WhiteoutForFile(filename)) && !diffEntries.Exists(filename) {
			
			// Determine whether the entry is a directory
			if details.IsDir() {
				
				// Merge the directory recursively
				errorChannels = append(errorChannels, apply.ApplyRecursive(filepath.Join(subpath, filename), details, false))
				
			} else {
				
				// DEBUG
				//log.Println("Merge file from base layer", filename)
				
				if err := apply.applyFile(apply.BaseDir, subpath, filename, details); err != nil {
					return err
				}
			}
		}
	}
	
	// Merge the contents of the diff into the output directory
	for filename, details := range diffEntries {
		
		// Ignore whiteout files
		if !IsWhiteout(filename) {
			
			// Determine whether the entry is a directory
			if details.IsDir() {
				
				// Merge the directory recursively, indicating whether the directory has been erased to ensure whiteouts propagate to subdirectories
				erased := ignoreBase || diffEntries.Exists(WhiteoutForFile(filename))
				errorChannels = append(errorChannels, apply.ApplyRecursive(filepath.Join(subpath, filename), details, erased))
				
			} else {
				
				// DEBUG
				//log.Println("Merge file from diff", filename)
				
				if err := apply.applyFile(apply.DiffDir, subpath, filename, details); err != nil {
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
	
	return aggregated
}

// Mirrors an individual file from either the base filesystem layer or the diff into the output directory
func (apply *DiffApplier) applyFile(origin string, subpath string, filename string, details fs.DirEntry) error {
	
	// Resolve the absolute paths to the source and target files
	source := filepath.Join(origin, subpath, filename)
	target := filepath.Join(apply.MergedDir, subpath, filename)
	
	// Attempt to mirror the file
	return MirrorFileWithAttributes(source, target, details)
}
