package configuration

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

// GlobalConfigurationPath returns the path of the YAML-based global
// configuration file. It does not verify that the file exists.
func GlobalConfigurationPath() (string, error) {
	// Compute the path to the user's home directory.
	homeDirectoryPath, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "unable to compute path to home directory")
	}

	// Success.
	return filepath.Join(homeDirectoryPath, filesystem.MutagenGlobalConfigurationName), nil
}
