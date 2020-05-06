package compose

import (
	"os"
	"os/exec"
)

// runCompose invokes Docker Compose with the specified arguments.
func runCompose(arguments *arguments) error {
	// Set up the command invocation.
	compose := exec.Command("docker-compose", arguments.reconstitute()...)

	// Set up streams.
	// TODO: Handle the case that configuration is piped from standard input.
	compose.Stdin = os.Stdin
	compose.Stdout = os.Stdout
	compose.Stderr = os.Stderr

	// TODO: Figure out signal handling. See what Docker Compose handles itself.

	// Run Docker Compose.
	return compose.Run()
}
