package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Determines if the specified file or directory exists
func Exists(path string) (bool) {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

type DirectoryEntryMap map[string]fs.DirEntry

func (m DirectoryEntryMap) Exists(filename string) bool {
	_, ok := m[filename]
	return ok
}

// Wraps `os.ReadDir()`, returning the list of directory entries as a map from basename to entry
func ReadDirAsMap(path string) (DirectoryEntryMap, error) {
	
	entryMap := make(DirectoryEntryMap)
	if !Exists(path) {
		return entryMap, nil
	}
	
	entryList, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	
	for _, entry := range entryList {
		entryMap[entry.Name()] = entry
	}
	
	return entryMap, nil
}

type LayerList struct {
	Layers []string `json:"layers"`
}

type LayerMerger struct {
	
	// The root directory for the base filesystem layer
	BaseDir string
	
	// The root directory for the diff to be applied against the base filesystem layer
	DiffDir string
	
	// The root directory in which to place the 
	MergedDir string
}

// Generates the filename for a whiteout file given the filename of the original file
func (merger *LayerMerger) whiteoutForFile(filename string) string {
	return fmt.Sprint(".wh.", filename)
}

// Determines whether the given filename represents a whiteout file or opaque whiteout file
func (merger *LayerMerger) isWhiteout(filename string) bool {
	return strings.HasPrefix(filename, ".wh.")
}

/*
 - Retrieve directory entries for the subpath joined with the base directory (if it exists and we didn't receive a boolean indicating that it was removed by a whiteout file) and create a map of filenames to file entry details
 - Do the same for the subpath joined with the diff directory (if it exists)
 - Check if there is an opaque whiteout file for the subpath in the diff entries. If so, skip iterating the base entries. (If there was a whiteout file or an opaque whiteout file in the parent directory that removed it then the list of entries will be empty anyway.)
 - If there was no opaque whiteout file, for each base entry, check if there is a matching diff entry that overwrites it or a whiteout file that removes it. If either exists then skip the entry, otherwise copy it over to the merged directory, calling the merge function recursively if the entry is a directory.
 - For each diff entry that is not a whiteout file, copy it over to the merged directory, calling the merge function recursively if the entry is a directory. If there was a whiteout file for the directory or an opaque whiteout file then pass true for the boolean that indicates the base version of the directory was removed and should thus be ignored.
*/

// Recursively applies the diff for a given filesystem subpath to the contents of the base filesystem layer
func (merger *LayerMerger) MergeRecursive(subpath string, whiteoutInParent bool) error {
	
	// DEBUG
	log.Println("Entering subpath", subpath)
	
	// List the directory contents for the subpath in the diff
	diffEntries, err := ReadDirAsMap(filepath.Join(merger.DiffDir, subpath))
	if err != nil {
		return err
	}
	
	// Determine whether the contents of the subpath from the base filesystem layer have been erased by a whiteout file or
	// opaque whiteout file in the parent directory of the diff or an opaque whiteout file the current directory of the diff
	haveOpaqueWhiteout := diffEntries.Exists(".wh..wh..opq")
	ignoreBase := whiteoutInParent || haveOpaqueWhiteout
	
	// Only list the directory contents for the subpath in the base filesystem layer if it has not been erased
	baseEntries := make(DirectoryEntryMap)
	if !ignoreBase {
		var err error
		baseEntries, err = ReadDirAsMap(filepath.Join(merger.BaseDir, subpath))
		if err != nil {
			return err
		}
	}
	
	// Merge the contents of the base filesystem layer into the output directory
	for filename, details := range baseEntries {
		
		// Ignore the file or directory if it has been erased or overwritten
		if !diffEntries.Exists(merger.whiteoutForFile(filename)) && !diffEntries.Exists(filename) {
			
			// If the entry is a directory then merge it recursively
			if details.IsDir() {
				if err := merger.MergeRecursive(filepath.Join(subpath, filename), false); err != nil {
					return err
				}
			} else {
				// DEBUG
				log.Println("Merge file from base layer", filename)
			}
		}
	}
	
	// Merge the contents of the diff into the output directory
	for filename, details := range diffEntries {
		
		// Ignore whiteout files
		if !merger.isWhiteout(filename) {
			
			// If the entry is a directory then merge it recursively
			// (Make sure we indicate whether the directory has been erased to ensure whiteouts propagate to subdirectories)
			if details.IsDir() {
				erased := ignoreBase || diffEntries.Exists(merger.whiteoutForFile(filename))
				if err := merger.MergeRecursive(filepath.Join(subpath, filename), erased); err != nil {
					return err
				}
			} else {
				// DEBUG
				log.Println("Merge file from diff", filename)
			}
		}
	}
	
	return nil
}

func main() {

	// Read the layers JSON file
	jsonBytes, err := os.ReadFile("../Sample layers/out/layers.json")
	if err != nil {
		log.Fatal(err)
	}

	// Parse the JSON data
	var layers LayerList
	if err := json.Unmarshal(jsonBytes, &layers); err != nil {
		log.Fatal(err)
	}

	// Iterate over the list of filesystem layers
	previousLayer := ""
	for _, layer := range layers.Layers {
		
		merger := &LayerMerger{
			BaseDir: filepath.Join("..", "Sample layers", "out", "layers", previousLayer, "merged"),
			DiffDir: filepath.Join("..", "Sample layers", "out", "layers", layer, "diff"),
			MergedDir: filepath.Join("..", "Sample layers", "out", "layers", layer, "merged"),
		}
		
		// Remove the merged directory if it already exists
		if Exists(merger.MergedDir) {
			if err := os.RemoveAll(merger.MergedDir); err != nil {
				log.Fatal(err)
			}
		}
		
		// Determine if this is the base layer
		if previousLayer == "" {

			// For the base layer, the merged directory is just a symlink to the diff directory
			if err := os.Symlink("./diff", merger.MergedDir); err != nil {
				log.Fatal(err)
			}

		} else {

			// Create the merged directory
			if err := os.Mkdir(merger.MergedDir, os.ModePerm); err != nil {
				log.Fatal(err)
			}

			// Apply the layer's diff to the merged contents of the previous layer
			log.Println("Apply diff", layer, "against base layer", previousLayer, "...")
			if err := merger.MergeRecursive("rootdir", false); err != nil {
				log.Fatal(err)
			}

		}

		/*
			// TEMPORARY: list the top-level contents of the layer's diff directory
			files, err := os.ReadDir("../Sample layers/out/layers/" + layer + "/diff")
			if err != nil {
				log.Fatal(err)
			}

			// TEMPORARY: list details of the top-level files and subdirectories
			log.Println("Contents of filesystem layer", layer)
			for _, file := range files {
				log.Println(" -", file.Name())
				if file.Type().IsDir() {
					log.Println("   Is a directory")
				} else if file.Type().IsRegular() {
					log.Println("   Is a regular file")
				} else if file.Type() == fs.ModeSymlink {
					log.Println("   Is a symlink")
				}
				log.Println("")
			}
		*/

		previousLayer = layer
	}
}
