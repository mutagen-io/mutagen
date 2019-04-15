package kubectl

import (
	"os/exec"

	"github.com/havoc-io/mutagen/pkg/process"
)

// commandSearchPaths specifies locations on macOS where we might find the
// kubectl binary.
var commandSearchPaths = []string{
	"/usr/local/bin",
}

// kubectlCommandForPlatform will search for a suitable kubectl command
// implementation on macOS.
func kubectlCommandForPlatform() (string, error) {
	// First, attempt to find the kubectl executable using the PATH environment
	// variable. If that works, use that result.
	if path, err := exec.LookPath("kubectl"); err == nil {
		return path, nil
	}

	// If the PATH-based lookup fails, attempt to search a set of common
	// locations where Kubectl installations reside on macOS. This is
	// unfortunately necessary due to launchd stripping almost everything out of
	// the PATH environment variable, including /usr/local/bin, the default
	// installation path for Kubectl for Mac.
	return process.FindCommand("kubectl", commandSearchPaths)
}
