package project

import (
	"os"
	"os/exec"
)

// runInShell runs the specified command using the system shell. On Windows
// systems, this is %COMSPEC% (with a fallback to cmd.exe if unspecified).
func runInShell(command string) error {
	// Determine the shell to use.
	shell := os.Getenv("COMSPEC")
	if shell == "" {
		shell = "cmd.exe"
	}

	// Set up the process.
	process := exec.Command(shell, "/c", command)
	process.Stdin = os.Stdin
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr

	// Run the process and wait for its completion.
	return process.Run()
}
