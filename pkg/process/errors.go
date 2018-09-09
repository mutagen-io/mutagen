package process

import (
	"strings"
)

const (
	windowsInvalidCommandFragment  = "is not recognized as an internal or external command"
	windowsCommandNotFoundFragment = "The system cannot find the path specified"
)

// OutputIsWindowsInvalidCommand returns whether or not a process' error output
// represents an invalid command error on Windows.
func OutputIsWindowsInvalidCommand(output string) bool {
	return strings.Contains(output, windowsInvalidCommandFragment)
}

// OutputIsWindowsCommandNotFound returns whether or not a process' error output
// represents a command not found error on Windows.
func OutputIsWindowsCommandNotFound(output string) bool {
	return strings.Contains(output, windowsCommandNotFoundFragment)
}
