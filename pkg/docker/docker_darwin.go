package docker

import (
	"os/exec"

	"github.com/mutagen-io/mutagen/pkg/process"
)

// commandSearchPaths specifies locations on macOS where we might find the
// docker binary.
var commandSearchPaths = []string{
	"/usr/local/bin",
}

// commandPathForPlatform will search for a suitable docker command
// implementation on macOS.
func commandPathForPlatform() (string, error) {
	// First, attempt to find the docker executable using the PATH environment
	// variable. If that works, use that result.
	if path, err := exec.LookPath("docker"); err == nil {
		return path, nil
	}

	// If the PATH-based lookup fails, attempt to search a set of common
	// locations where Docker installations reside on macOS. This is
	// unfortunately necessary due to launchd stripping almost everything out of
	// the PATH environment variable, including /usr/local/bin, the default
	// installation path for Docker for Mac. This fallback is only necessary in
	// the context of running the Mutagen daemon as a launchd service, so it's
	// not necessary for (say) Docker Compose, which isn't used by the daemon.
	return process.FindCommand("docker", commandSearchPaths)
}
