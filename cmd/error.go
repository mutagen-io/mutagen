package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

// Warning prints a warning message to standard error.
func Warning(message string) {
	fmt.Fprintln(color.Error, color.YellowString("Warning:"), message)
}

// Error prints an error message to standard error.
func Error(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
}

// Fatal prints an error message to standard error and then terminates the
// process with an error exit code.
func Fatal(err error) {
	Error(err)
	os.Exit(1)
}
