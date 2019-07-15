package configuration

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
)

// YAMLConfiguration is the global YAML configuration object type.
type YAMLConfiguration struct {
	// Forwarding is the global forwarding configuration.
	Forwarding forwarding.YAMLConfiguration `yaml:"forwarding"`
	// Synchronization is the global forwarding configuration.
	Synchronization synchronization.YAMLConfiguration `yaml:"sync"`
}

// Load attempts to load a YAML-based Mutagen configuration file from the
// specified path. If the path is empty, then the global Mutagen configuration
// file, if any, will be loaded. If the file doesn't exist, an all-default
// configuration object is returned.
func Load(path string) (*YAMLConfiguration, error) {
	// If path is empty, compute the path to the global configuration file.
	if path == "" {
		// Compute the path to the user's home directory.
		homeDirectory, err := os.UserHomeDir()
		if err != nil {
			return nil, errors.Wrap(err, "unable to compute path to home directory")
		}

		// Check if the legacy global configuration file exists. If so, warn the
		// user that they need to convert to YAML.
		legacyGlobalConfigurationPath := filepath.Join(
			homeDirectory,
			filesystem.MutagenLegacyGlobalConfigurationName,
		)
		if _, err := os.Stat(legacyGlobalConfigurationPath); err == nil {
			return nil, errors.New("please convert your TOML-based configuration to YAML (sorry!)")
		}

		// Compute the path to the global configuration file.
		path = filepath.Join(homeDirectory, filesystem.MutagenGlobalConfigurationName)
	}

	// Attempt to load.
	result := &YAMLConfiguration{}
	if err := encoding.LoadAndUnmarshalYAML(path, result); err != nil {
		if os.IsNotExist(err) {
			return &YAMLConfiguration{}, nil
		}
		return nil, errors.Wrap(err, "unable to load configuration file")
	}

	// Success.
	return result, nil
}
