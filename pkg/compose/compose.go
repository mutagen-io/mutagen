package compose

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/mutagen-io/mutagen/pkg/process"
)

// CommandPath returns the absolute path specification to use for invoking
// Docker Compose. It will use the MUTAGEN_DOCKER_COMPOSE_PATH environment
// variable if provided, otherwise falling back to a standard os/exec.LookPath
// implementation.
func CommandPath() (string, error) {
	// If MUTAGEN_DOCKER_COMPOSE_PATH is specified, then use it to perform the
	// lookup.
	if searchPath := os.Getenv("MUTAGEN_DOCKER_COMPOSE_PATH"); searchPath != "" {
		return process.FindCommand("docker-compose", []string{searchPath})
	}

	// Otherwise fall back to a standard search of the user's path.
	return exec.LookPath("docker-compose")
}

// Command prepares (but does not start) a Docker Compose command with the
// specified arguments and scoped to lifetime of the provided context.
func Command(context context.Context, args ...string) (*exec.Cmd, error) {
	// Identify the command path.
	commandPath, err := CommandPath()
	if err != nil {
		return nil, fmt.Errorf("unable to identify 'docker-compose' command: %w", err)
	}

	// Create the command.
	return exec.CommandContext(context, commandPath, args...), nil
}
