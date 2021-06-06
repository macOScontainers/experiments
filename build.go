// +build never

package main

import (
	"flag"
	"os"

	module "github.com/tensorworks/go-build-helpers/pkg/module"
	validation "github.com/tensorworks/go-build-helpers/pkg/validation"
)

// Alias validation.ExitIfError() as check()
var check = validation.ExitIfError

func main() {
	
	// Parse our command-line flags
	doClean := flag.Bool("clean", false, "cleans build outputs")
	flag.Parse()
	
	// Create a build helper for the Go module
	mod, err := module.ModuleInCwd()
	check(err)
	
	// Determine if we're cleaning the build outputs
	if *doClean == true {
		check( mod.CleanAll() )
		os.Exit(0)
	}
	
	// Build our binaries
	check( mod.BuildBinariesForHost(module.DefaultBinDir, module.Undecorated) )
}
