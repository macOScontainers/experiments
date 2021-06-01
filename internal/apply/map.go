package apply

import (
	"io/fs"
	"os"

	"github.com/macoscontainers/experiments/internal/filesystem"
)

// Represents a mapping from filenames to directory entry details
type DirEntryMap map[string]fs.DirEntry

// Determines if the specified entry exists in the map
func (m DirEntryMap) Exists(filename string) bool {
	_, ok := m[filename]
	return ok
}

// Wraps `os.ReadDir()` to generate a new DirEntryMap object
func ReadDirAsMap(path string) (DirEntryMap, error) {
	
	// If the specified directory does not exist then return an empty map
	entryMap := make(DirEntryMap)
	if !filesystem.Exists(path) {
		return entryMap, nil
	}
	
	// Attempt to list the contents of the specified directory
	entryList, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	
	// Build our map
	for _, entry := range entryList {
		entryMap[entry.Name()] = entry
	}
	
	return entryMap, nil
}
