package kubectl

import (
	"os"

	"github.com/havoc-io/mutagen/pkg/process"
)

// kubectlCommand returns the name or path specification to use for invoking
// Kubectl. It will use the MUTAGEN_KUBECTL_PATH environment variable if provided,
// otherwise falling back to a platform-specific implementation.
func kubectlCommand() (string, error) {
	// If MUTAGEN_KUBECTL_PATH is specified, then use it to perform the lookup.
	if searchPath := os.Getenv("MUTAGEN_KUBECTL_PATH"); searchPath != "" {
		return process.FindCommand("kubectl", []string{searchPath})
	}

	// Otherwise fall back to the platform-specific implementation.
	return kubectlCommandForPlatform()
}
