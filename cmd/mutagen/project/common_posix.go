// +build !windows

package project

import (
	"os"
	"os/exec"
)

// runInShell runs the specified command using the system shell. On POSIX
// systems, this is /bin/sh.
func runInShell(command string) error {
	// Set up the process.
	process := exec.Command("/bin/sh", "-c", command)
	process.Stdin = os.Stdin
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr

	// Run the process and wait for its completion.
	return process.Run()
}
