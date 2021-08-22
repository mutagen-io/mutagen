package global

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

// ConfigurationPath returns the path of the YAML-based global configuration
// file. It does not verify that the file exists.
func ConfigurationPath() (string, error) {
	// Compute the path to the user's home directory.
	homeDirectoryPath, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to compute path to home directory: %w", err)
	}

	// Success.
	return filepath.Join(homeDirectoryPath, filesystem.MutagenGlobalConfigurationName), nil
}
