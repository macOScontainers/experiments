// +build darwin

package filesystem

// #include <stdlib.h>
// #include <string.h>
// #include <sys/attr.h>
// #include <sys/clonefile.h>
import "C"
import (
	"unsafe"
)

// Flags for use with Clone()
const (
	
	// Don't follow symbolic links
	CLONE_NOFOLLOW    = 0x0001
	
	// Don't copy ownership information from source
	CLONE_NOOWNERCOPY = 0x0002
)

func Clone(source string, destination string, flags uint32) error {
	
	// Convert the source path to a null terminated C string
	sourceCstr := C.CString(source)
	defer C.free(unsafe.Pointer(sourceCstr))
	
	// Convert the destination path to a null terminated C string
	destinationCstr := C.CString(destination)
	defer C.free(unsafe.Pointer(destinationCstr))
	
	// Attempt to clone the source file or directory to the destination
	result, err := C.clonefile(sourceCstr, destinationCstr, C.uint(flags))
	if result != 0 {
		return err
	}
	
	return nil
}
