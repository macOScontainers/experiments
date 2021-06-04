package layer

import (
	"fmt"
	"strings"
)

// The prefix used to specify that a file is a whiteout file
const WHITEOUT_FILENAME_PREFIX = ".wh."

// The filename used to specify that a file is an opaque whiteout file
const OPAQUE_WHITEOUT_FILENAME = ".wh..wh..opq"

// Determines whether the given filename represents a whiteout file or opaque whiteout file
func IsWhiteout(filename string) bool {
	return strings.HasPrefix(filename, WHITEOUT_FILENAME_PREFIX)
}

// Generates the filename for a whiteout file given the filename of the original file
func WhiteoutForFile(filename string) string {
	return fmt.Sprint(WHITEOUT_FILENAME_PREFIX, filename)
}
