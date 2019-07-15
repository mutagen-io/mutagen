package docker

import (
	"context"
	"os"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/process"
)

// CommandPath returns the absolute path specification to use for invoking
// Docker. It will use the MUTAGEN_DOCKER_PATH environment variable if provided,
// otherwise falling back to a platform-specific implementation.
func CommandPath() (string, error) {
	// If MUTAGEN_DOCKER_PATH is specified, then use it to perform the lookup.
	if searchPath := os.Getenv("MUTAGEN_DOCKER_PATH"); searchPath != "" {
		return process.FindCommand("docker", []string{searchPath})
	}

	// Otherwise fall back to the platform-specific implementation.
	return commandPathForPlatform()
}

// Command prepares (but does not start) a Docker command with the specified
// arguments and scoped to lifetime of the provided context.
func Command(context context.Context, args ...string) (*exec.Cmd, error) {
	// Identify the command path.
	commandPath, err := CommandPath()
	if err != nil {
		return nil, errors.Wrap(err, "unable to identify 'docker' command")
	}

	// Create the command.
	return exec.CommandContext(context, commandPath, args...), nil
}
