package configuration

import (
	"os"

	"github.com/havoc-io/mutagen/pkg/encoding"
	"github.com/havoc-io/mutagen/pkg/filesystem"
)

type Configuration struct {
	Ignore struct {
		Default []string `toml:"default"`
	} `toml:"ignore"`
}

func Load() (*Configuration, error) {
	// Create a configuration that we can decode into. We set any default values
	// here because nothing will be modified in this structure if the
	// configuration file doesn't exist.
	result := &Configuration{}

	// Attempt to load the configuration from disk.
	if err := encoding.LoadAndUnmarshalTOML(filesystem.MutagenConfigurationPath, result); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	// Return the configuration.
	return result, nil
}
