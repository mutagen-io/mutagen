package project

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

// runInShell runs the specified command using the system shell. On Windows
// systems, this is %ComSpec% (with a fallback to a fully qualified cmd.exe if
// %ComSpec% is not an absolute path (which includes cases where it's empty)).
func runInShell(command string) error {
	// Determine the shell to use.
	shell := os.Getenv("ComSpec")
	if !filepath.IsAbs(shell) {
		systemRoot := os.Getenv("SystemRoot")
		if !filepath.IsAbs(systemRoot) {
			return errors.New("invalid ComSpec and SystemRoot environment variables")
		}
		shell = filepath.Join(systemRoot, "System32", "cmd.exe")
	}

	// Set up the process.
	process := exec.Command(shell, "/c", command)
	process.Stdin = os.Stdin
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr

	// Run the process and wait for its completion.
	return process.Run()
}
