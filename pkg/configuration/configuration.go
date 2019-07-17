package configuration

import (
	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
)

// YAMLConfiguration is the global YAML configuration object type.
type YAMLConfiguration struct {
	// Forwarding is the global forwarding configuration.
	Forwarding struct {
		// Defaults are the global forwarding configuration defaults.
		Defaults forwarding.YAMLConfiguration `yaml:"defaults"`
	} `yaml:"forward"`
	// Synchronization is the global synchronization configuration.
	Synchronization struct {
		// Defaults are the global synchronization configuration defaults.
		Defaults synchronization.YAMLConfiguration `yaml:"defaults"`
	} `yaml:"sync"`
}

// Load attempts to load a YAML-based Mutagen configuration file from the
// specified path.
func Load(path string) (*YAMLConfiguration, error) {
	// Create the target configuration object.
	result := &YAMLConfiguration{}

	// Attempt to load. We pass-through os.IsNotExist errors.
	if err := encoding.LoadAndUnmarshalYAML(path, result); err != nil {
		return nil, err
	}

	// Success.
	return result, nil
}
