package layer

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

// Copies the attributes of the source file or directory to the target file or directory
func CopyAttributes(source string, target string, details fs.DirEntry) error {
	
	// Retrieve the attributes from the DirEntry object
	info, err := details.Info()
	if err != nil {
		return err
	}
	
	// Extract the Unix-specific attributes
	sys, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("fs.FileInfo.Sys() was not a syscall.Stat_t object")
	}
	
	// Copy permissions
	if details.Type() != fs.ModeSymlink {
		if err := os.Chmod(target, info.Mode()); err != nil {
			return err
		}
	}
	
	// Copy ownership information
	return os.Lchown(target, int(sys.Uid), int(sys.Gid))
}

// Mirrors the source file in the target location and preserves its attributes
// (Note that this function uses hardlinks where possible to avoid duplicating data)
func MirrorFileWithAttributes(source string, target string, details fs.DirEntry) error {
	
	// Determine what type of file we are mirroring
	switch details.Type() {
	
	// Preserve symlinks
	case fs.ModeSymlink:
		
		// Read the contents of the symlink
		link, err := os.Readlink(source)
		if err != nil {
			return err
		}
		
		// Recreate the symlink in the target location
		if err := os.Symlink(link, target); err != nil {
			return err
		}
		
		// Copy over the attributes of the symlink
		return CopyAttributes(source, target, details)
		
	// Hardlink all other types of files
	default:
		return os.Link(source, target)
	}
}
