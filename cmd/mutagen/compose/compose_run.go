package compose

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/mutagen-io/mutagen/cmd"
)

// compose invokes Docker Compose with the specified top-level flags, command
// name, and arguments. It forward standard input/output/error to the child
// process and terminates the current process with the same exit code as the
// child process. If an error occurs while trying to invoke Docker Compose, then
// this function will print an error message and terminate the current process
// with an error exit code. If command is an empty string, then no command is
// specified to Docker Compose and arguments are ignored (though top-level flags
// are still included in the Docker Compose invocation).
func compose(topLevelFlags []string, command string, arguments []string) {
	// TODO: Figure out if there's any signal handling that we need to set up.
	// Docker Compose has a bunch of internal signal handling at its entry
	// point, but this may not be necessary with the Go runtime in the same way
	// that it is with the Python runtime. In any case, we'll likely need to
	// forward signals.

	// Set up the Docker Compose commmand.
	composeArguments := make([]string, 0, len(topLevelFlags)+1+len(arguments))
	composeArguments = append(composeArguments, topLevelFlags...)
	if command != "" {
		composeArguments = append(composeArguments, command)
		composeArguments = append(composeArguments, arguments...)
	}
	compose := exec.Command("docker-compose", composeArguments...)

	// Setup input and output streams.
	compose.Stdin = os.Stdin
	compose.Stdout = os.Stdout
	compose.Stderr = os.Stderr

	// Run Docker Compose.
	if err := compose.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitCode := exitErr.ExitCode(); exitCode < 1 {
				os.Exit(1)
			} else {
				os.Exit(exitCode)
			}
		} else {
			cmd.Fatal(fmt.Errorf("unable to invoke Docker Compose: %w", err))
		}
	}

	// Success.
	os.Exit(0)
}
