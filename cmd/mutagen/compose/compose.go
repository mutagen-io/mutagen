package compose

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/mutagen-io/mutagen/cmd"
)

// compose invokes Docker Compose with the specified arguments, environment,
// standard input, and exit behavior. If environment is nil, then the Docker
// Compose process will inherit the current environment. If input is nil, then
// Docker Compose will read from the null device (os.DevNull). If an error
// occurs while attempting to invoke Docker Compose, then this function will
// print an error message and terminate the current process with an exit code of
// 1. If invocation succeeds but Docker Compose exits with a non-0 exit code,
// then this function won't print an error message but will terminate the
// current process with a matching exit code. If invocation succeeds and Docker
// Compose exits with an exit code of 0, then this function will simply return,
// unless exitOnSuccess is specified, in which case this process will terminate
// the current process with an exit code of 0.
func compose(arguments []string, environment map[string]string, input io.Reader, exitOnSuccess bool) {
	// Create the command.
	compose := exec.Command("docker-compose", arguments...)

	// Set up the command environment.
	if environment != nil {
		compose.Env = make([]string, 0, len(environment))
		for k, v := range environment {
			compose.Env = append(compose.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Set up input and output streams.
	compose.Stdin = input
	compose.Stdout = os.Stdout
	compose.Stderr = os.Stderr

	// Run the command.
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

	// Terminate the current process if necessary.
	if exitOnSuccess {
		os.Exit(0)
	}
}
