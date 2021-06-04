// +build darwin

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/macoscontainers/experiments/internal/filesystem"
)


func main() {
	
	// Parse our command-line flags
	flag.Parse()
	if len(flag.Args()) < 2 {
		fmt.Println("Usage: clone <SOURCE> <DEST>")
		os.Exit(0)
	}
	
	// Retrieve the source and destination paths
	source := flag.Args()[0]
	dest := flag.Args()[1]
	
	// Attempt to perform the clone
	if err := filesystem.Clone(source, dest, filesystem.CLONE_NOFOLLOW); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	} else {
		fmt.Println("Clone successful.")
	}
}
