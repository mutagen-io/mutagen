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

// loadFromPath is the internal loading function. We keep it separate from Load
// so that we can get full test coverage using temporary files.
func loadFromPath(path string) (*Configuration, error) {
	// Create a configuration that we can decode into. We set any default values
	// here because nothing will be modified in this structure if the
	// configuration file doesn't exist.
	result := &Configuration{}

	// Attempt to load the configuration from disk.
	// TODO: Should we implement a caching mechanism where we run a stat call
	// and watch for filesystem modification?
	if err := encoding.LoadAndUnmarshalTOML(path, result); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	// Return the configuration.
	return result, nil
}

// Load loads the Mutagen configuration file from disk and populates a
// Configuration structure. If the Mutagen configuration file does not exist,
// this method will return a structure with the default configuration values.
// The returned structure is not re-used, so its members can be freely mutated.
func Load() (*Configuration, error) {
	return loadFromPath(filesystem.MutagenConfigurationPath)
}
