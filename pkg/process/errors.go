package process

import (
	"strings"
)

const (
	// posixCommandNotFoundFragment is a fragment of the error output returned
	// on some POSIX shells when a command is not found. The capitalization of
	// the word "command" is inconsistent between shells, so only part of the
	// word is used.
	posixCommandNotFoundFragment = "ommand not found"
	// windowsInvalidCommandFragment is a fragment of the error output returned
	// on Windows systems when a command is not recognized.
	windowsInvalidCommandFragment = "is not recognized as an internal or external command"
	// windowsCommandNotFoundFragment is a fragment of the error output returned
	// on Windows systems when a command cannot be found.
	windowsCommandNotFoundFragment = "The system cannot find the path specified"
)

// OutputIsPOSIXCommandNotFound returns whether or not a process' error output
// represents a command not found error on POSIX systems.
func OutputIsPOSIXCommandNotFound(output string) bool {
	return strings.Contains(output, posixCommandNotFoundFragment)
}

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
