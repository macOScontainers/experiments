package apply

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
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

// Determines whether the given filename represents a whiteout file or opaque whiteout file
func (applier *DiffApplier) isWhiteout(filename string) bool {
	return strings.HasPrefix(filename, WHITEOUT_FILENAME_PREFIX)
}

// Generates the filename for a whiteout file given the filename of the original file
func (applier *DiffApplier) whiteoutForFile(filename string) string {
	return fmt.Sprint(WHITEOUT_FILENAME_PREFIX, filename)
}

// Recursively applies the diff for a given filesystem subpath to the contents of the base filesystem layer
func (applier *DiffApplier) ApplyRecursive(subpath string, whiteoutInParent bool) error {
	
	// DEBUG
	log.Println("Entering subpath", subpath)
	
	// Unless this is the root directory, create the appropriate subdirectory in the merged output directory
	if subpath != "" {
		if err := os.MkdirAll(filepath.Join(applier.MergedDir, subpath), os.ModePerm); err != nil {
			return err
		}
	}
	
	// List the directory contents for the subpath in the diff
	diffEntries, err := ReadDirAsMap(filepath.Join(applier.DiffDir, subpath))
	if err != nil {
		return err
	}
	
	// Determine whether the contents of the subpath from the base filesystem layer have been erased by a whiteout file or
	// opaque whiteout file in the parent directory of the diff or an opaque whiteout file the current directory of the diff
	haveOpaqueWhiteout := diffEntries.Exists(OPAQUE_WHITEOUT_FILENAME)
	ignoreBase := whiteoutInParent || haveOpaqueWhiteout
	
	// Only list the directory contents for the subpath in the base filesystem layer if it has not been erased
	baseEntries := make(DirEntryMap)
	if !ignoreBase {
		var err error
		baseEntries, err = ReadDirAsMap(filepath.Join(applier.BaseDir, subpath))
		if err != nil {
			return err
		}
	}
	
	// Merge the contents of the base filesystem layer into the output directory
	for filename, details := range baseEntries {
		
		// Ignore the file or directory if it has been erased or overwritten
		if !diffEntries.Exists(applier.whiteoutForFile(filename)) && !diffEntries.Exists(filename) {
			
			// If the entry is a directory then merge it recursively
			if details.IsDir() {
				if err := applier.ApplyRecursive(filepath.Join(subpath, filename), false); err != nil {
					return err
				}
			} else {
				
				// DEBUG
				log.Println("Merge file from base layer", filename)
				
				if err := applier.applyFile(applier.BaseDir, subpath, filename, details); err != nil {
					return err
				}
			}
		}
	}
	
	// Merge the contents of the diff into the output directory
	for filename, details := range diffEntries {
		
		// Ignore whiteout files
		if !applier.isWhiteout(filename) {
			
			// If the entry is a directory then merge it recursively
			// (Note that we indicate whether the directory has been erased to ensure whiteouts propagate to subdirectories)
			if details.IsDir() {
				erased := ignoreBase || diffEntries.Exists(applier.whiteoutForFile(filename))
				if err := applier.ApplyRecursive(filepath.Join(subpath, filename), erased); err != nil {
					return err
				}
			} else {
				
				// DEBUG
				log.Println("Merge file from diff", filename)
				
				if err := applier.applyFile(applier.DiffDir, subpath, filename, details); err != nil {
					return err
				}
			}
		}
	}
	
	return nil
}

// Copies an individual file from either the base filesystem layer or the diff into the output directory
func (applier *DiffApplier) applyFile(origin string, subpath string, filename string, details fs.DirEntry) error {
	
	// Resolve the absolute paths to the source and target files
	source := filepath.Join(origin, subpath, filename)
	target := filepath.Join(applier.MergedDir, subpath, filename)
	
	// Determine what type of file we are copying
	switch details.Type() {
	
	// Preserve symlinks
	case fs.ModeSymlink:
		
		// Read the contents of the symlink
		link, err := os.Readlink(source)
		if err != nil {
			return err
		}
		
		// Recreate the symlink in the target location
		return os.Symlink(link, target)
		
	// Hardlink all other types of files
	default:
		return os.Link(source, target)
	}
}
