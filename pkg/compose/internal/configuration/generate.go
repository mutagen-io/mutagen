package configuration

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// GeneratedServiceConfiguration encodes a subset of the Docker Compose service
// configuration format. It is only used for generating Docker Compose service
// configurations, and thus only encodes those fields needed by generated
// Mutagen services. It is designed to be compatible with both 2.x and 3.x
// Docker Compose formats.
type GeneratedServiceConfiguration struct {
	// Image is the image for the service.
	Image string `yaml:"image"`
	// Networks are the network dependencies for the service.
	Networks []string `yaml:"networks,omitempty"`
	// Volumes are the volume dependencies for the service.
	Volumes []string `yaml:"volumes,omitempty"`
}

// GeneratedComposeConfiguration encodes a subset of the Docker Compose
// configuration format. It is only used for generating configuration files and
// thus only encodes those fields needed by Mutagen services. It is designed to
// be compatible with both 2.x and 3.x Docker Compose configuration formats.
type GeneratedComposeConfiguration struct {
	// Version is the Docker Compose configuration file version.
	Version string `yaml:"version"`
	// Services are the Docker Compose services.
	Services map[string]*GeneratedServiceConfiguration `yaml:"services"`
}

// Store encodes the configuration to YAML and writes it to a file at the
// specified path. The file is created with user-only permissions and must not
// already exist. The output file may be created even in the case of failure
// (for example if an error occurs during YAML encoding).
func (c *GeneratedComposeConfiguration) Store(path string) error {
	// Open the output file and defer its closure.
	output, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("unable to open output file: %w", err)
	}
	defer output.Close()

	// Create a YAML encoder and defer its closure.
	encoder := yaml.NewEncoder(output)
	defer encoder.Close()

	// Perform encoding.
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("unable to encode configuration: %w", err)
	}

	// Success.
	return nil
}
