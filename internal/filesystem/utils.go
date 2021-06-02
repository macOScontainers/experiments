package filesystem

import "os"

// Determines if the specified file or directory exists
func Exists(path string) (bool) {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}
