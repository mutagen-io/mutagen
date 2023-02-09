package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/mutagen-io/mutagen/pkg/platform"
)

// CommandPath returns the absolute path specification to use for invoking
// Docker. It will use the MUTAGEN_DOCKER_PATH environment variable if provided,
// otherwise falling back to a platform-specific implementation.
func CommandPath() (string, error) {
	// If MUTAGEN_DOCKER_PATH is specified, then use it to perform the lookup.
	if searchPath := os.Getenv("MUTAGEN_DOCKER_PATH"); searchPath != "" {
		return platform.FindCommand("docker", []string{searchPath})
	}

	// Otherwise fall back to the platform-specific implementation.
	return commandPathForPlatform()
}

// Command prepares (but does not start) a Docker command with the specified
// arguments and scoped to lifetime of the provided context.
func Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
	// Identify the command path.
	commandPath, err := CommandPath()
	if err != nil {
		return nil, fmt.Errorf("unable to identify 'docker' command: %w", err)
	}

	// Create the command.
	return exec.CommandContext(ctx, commandPath, args...), nil
}
