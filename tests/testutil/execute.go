package testutil

import (
	"log"
	"os"
	"os/exec"
)

// Prints and executes a command, directing its output to the output streams of the parent process
func Run(command ...string) error {
	
	// Prepare the command and direct stdout and stderr to the streams of the parent process
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// Log the command details and run the child process
	log.Println(command)
	return cmd.Run()
}
