package docker

import (
	"os"

	"github.com/havoc-io/mutagen/pkg/process"
)

// dockerCommand returns the name or path specification to use for invoking
// Docker. It will use the MUTAGEN_DOCKER_PATH environment variable if provided,
// otherwise falling back to a platform-specific implementation.
func dockerCommand() (string, error) {
	// If MUTAGEN_DOCKER_PATH is specified, then use it to perform the lookup.
	if searchPath := os.Getenv("MUTAGEN_DOCKER_PATH"); searchPath != "" {
		return process.FindCommand("docker", []string{searchPath})
	}

	// Otherwise fall back to the platform-specific implementation.
	return dockerCommandForPlatform()
}
