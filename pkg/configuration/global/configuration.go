package global

import (
	"github.com/mutagen-io/mutagen/pkg/configuration/forwarding"
	"github.com/mutagen-io/mutagen/pkg/configuration/synchronization"
	"github.com/mutagen-io/mutagen/pkg/encoding"
)

// Configuration is the global YAML configuration object type.
type Configuration struct {
	// Forwarding is the global forwarding configuration.
	Forwarding struct {
		// Defaults are the global forwarding configuration defaults.
		Defaults forwarding.Configuration `yaml:"defaults"`
	} `yaml:"forward"`
	// Synchronization is the global synchronization configuration.
	Synchronization struct {
		// Defaults are the global synchronization configuration defaults.
		Defaults synchronization.Configuration `yaml:"defaults"`
	} `yaml:"sync"`
}

// LoadConfiguration attempts to load a YAML-based Mutagen global configuration
// file from the specified path.
func LoadConfiguration(path string) (*Configuration, error) {
	// Create the target configuration object.
	result := &Configuration{}

	// Attempt to load. We pass-through os.IsNotExist errors.
	if err := encoding.LoadAndUnmarshalYAML(path, result); err != nil {
		return nil, err
	}

	// Success.
	return result, nil
}
