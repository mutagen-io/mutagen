package cmd

import (
	"os"
	"os/exec"

	"github.com/pkg/errors"

	isatty "github.com/mattn/go-isatty"
)

// HandleTerminalCompatibility automatically restarts the current process inside
// a terminal compatibility emulator if necessary. It currently only handles the
// case of mintty consoles on Windows requiring a relaunch of the current
// command inside winpty.
func HandleTerminalCompatibility() {
	// If we're not running inside a mintty-based terminal, then there's nothing
	// that we need to do.
	if !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return
	}

	// Since we're running inside a mintty-based terminal, we need to relaunch
	// using winpty, so first attempt to locate it.
	winpty, err := exec.LookPath("winpty")
	if err != nil {
		Fatal(errors.New("running inside mintty terminal and unable to locate winpty"))
	}

	// Compute the path to the current executable.
	executable, err := os.Executable()
	if err != nil {
		Fatal(errors.Wrap(err, "running inside mintty terminal and unable to locate current executable"))
	}

	// Build the argument list for winpty.
	arguments := make([]string, 0, len(os.Args))
	arguments = append(arguments, executable)
	arguments = append(arguments, os.Args[1:]...)

	// Create the command that we'll run.
	command := exec.Command(winpty, arguments...)

	// Set up its input/output streams.
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	// Run the command and terminate with its exit code.
	command.Run()
	os.Exit(command.ProcessState.ExitCode())
}
