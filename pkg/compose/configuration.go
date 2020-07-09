package compose

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/mutagen-io/mutagen/pkg/configuration/forwarding"
	"github.com/mutagen-io/mutagen/pkg/configuration/synchronization"
)

// forwardingConfiguration encodes a full forwarding session specification.
type forwardingConfiguration struct {
	// Source is the source URL for the session.
	Source string `yaml:"source"`
	// Destination is the destination URL for the session.
	Destination string `yaml:"destination"`
	// Configuration is the configuration for the session.
	Configuration forwarding.Configuration `yaml:",inline"`
	// ConfigurationSource is the source-specific configuration for the session.
	ConfigurationSource forwarding.Configuration `yaml:"configurationSource"`
	// ConfigurationDestination is the destination-specific configuration for
	// the session.
	ConfigurationDestination forwarding.Configuration `yaml:"configurationDestination"`
}

// synchronizationConfiguration encodes a full synchronization session
// specification.
type synchronizationConfiguration struct {
	// Alpha is the alpha URL for the session.
	Alpha string `yaml:"alpha"`
	// Beta is the beta URL for the session.
	Beta string `yaml:"beta"`
	// Configuration is the configuration for the session.
	Configuration synchronization.Configuration `yaml:",inline"`
	// ConfigurationAlpha is the alpha-specific configuration for the session.
	ConfigurationAlpha synchronization.Configuration `yaml:"configurationAlpha"`
	// ConfigurationBeta is the beta-specific configuration for the session.
	ConfigurationBeta synchronization.Configuration `yaml:"configurationBeta"`
}

// mutagenConfiguration encodes collections of Mutagen forwarding and
// synchronization sessions found under an "x-mutagen" extension field.
type mutagenConfiguration struct {
	// Forwarding represents the forwarding sessions to be created. If a
	// "defaults" key is present, it is treated as a template upon which other
	// configurations are layered, thus keeping syntactic compatibility with the
	// global Mutagen configuration file.
	Forwarding map[string]forwardingConfiguration `yaml:"forward"`
	// Synchronization represents the forwarding sessions to be created. If a
	// "defaults" key is present, it is treated as a template upon which other
	// configurations are layered, thus keeping syntactic compatibility with the
	// global Mutagen configuration file.
	Synchronization map[string]synchronizationConfiguration `yaml:"sync"`
}

// configuration represents a single Docker Compose configuration file.
type configuration struct {
	// Version is the configuration file schema version.
	Version string `yaml:"version"`
	// Services are the services defined in the file.
	Services map[string]yaml.Node `yaml:"services"`
	// Volumes are the volumes defined in the file.
	Volumes map[string]yaml.Node `yaml:"volumes"`
	// Networks are the networks defined in the file.
	Networks map[string]yaml.Node `yaml:"networks"`
	// XMutagen is the raw Mutagen configuration defined in the file.
	XMutagen yaml.Node `yaml:"x-mutagen"`
	// mutagen is the is the fully decoded Mutagen configuration derived from
	// the raw Mutagen configuration. The session specifications in this
	// configuration are not validated by loadConfiguration.
	mutagen mutagenConfiguration
}

// loadConfiguration reads, interpolates, and decodes a Docker Compose YAML
// configuration from the specified file. If the file contains multiple YAML
// documents, then only the first will be read. Interpolation is performed using
// the specified variable mapping.
func loadConfiguration(path string, variables map[string]string) (*configuration, error) {
	// Open the file and defer its closure.
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open configuration file: %w", err)
	}
	defer file.Close()

	// Wrap the file in a YAML decoder.
	decoder := yaml.NewDecoder(file)

	// Perform a generic decoding operation.
	var root yaml.Node
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("unable to parse YAML: %w", err)
	}

	// Perform interpolation.
	mapping := func(key string) (string, bool) {
		value, ok := variables[key]
		return value, ok
	}
	if err := interpolateYAML(&root, mapping); err != nil {
		return nil, fmt.Errorf("variable interpolation failed: %w", err)
	}

	// Decode the document into a more concrete configuration structure so that
	// we can extract and validate the Mutagen configuration.
	result := &configuration{}
	if err := root.Decode(result); err != nil {
		return nil, fmt.Errorf("unable to parse configuration file: %w", err)
	}

	// If there was no top-level "x-mutagen" specification, then we're done.
	if result.XMutagen.IsZero() {
		return result, nil
	}

	// Extract and re-serialize the interpolated "x-mutagen" section so that we
	// can perform strict decoding. We have to wrap the extracted section in a
	// document node for serialization to work.
	//
	// TODO: Once go-yaml/yaml#460 is resolved, we won't need to perform this
	// re-serialization since we'll be able to perform a strict decoding into
	// the final structure directly from the decoded YAML node. This may be
	// implemented by go-yaml/yaml#557.
	mutagenYAML, err := yaml.Marshal(&yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{&result.XMutagen},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to re-serialize Mutagen YAML: %w", err)
	}

	// Now re-parse that YAML with strict decoding to ensure that it's correct.
	decoder = yaml.NewDecoder(bytes.NewReader(mutagenYAML))
	decoder.KnownFields(true)
	if err := decoder.Decode(&result.mutagen); err != nil {
		return nil, fmt.Errorf("strict parsing of Mutagen YAML failed: %w", err)
	}

	// Success.
	return result, nil
}

// generatedServiceConfiguration encodes a subset of the Docker Compose service
// configuration format. It is only used for generating Docker Compose service
// configurations, and thus only encodes those fields needed by generated
// Mutagen services. It is designed to be compatible with both 2.x and 3.x
// Docker Compose formats.
type generatedServiceConfiguration struct {
	// Image is the image for the service.
	Image string `yaml:"image"`
	// Networks are the network dependencies for the service.
	Networks []string `yaml:"networks,omitempty"`
	// Volumes are the volume dependencies for the service.
	Volumes []string `yaml:"volumes,omitempty"`
}

// generatedComposeConfiguration encodes a subset of the Docker Compose
// configuration format. It is only used for generating configuration files and
// thus only encodes those fields needed by Mutagen services. It is designed to
// be compatible with both 2.x and 3.x Docker Compose configuration formats.
type generatedComposeConfiguration struct {
	// Version is the Docker Compose configuration file version.
	Version string `yaml:"version"`
	// Services are the Docker Compose services.
	Services map[string]*generatedServiceConfiguration `yaml:"services"`
}

// store encodes the configuration to YAML and writes it to a file at the
// specified path. The file is created with user-only permissions and must not
// already exist. The output file may be created even in the case of failure
// (for example if an error occurs during YAML encoding).
func (c *generatedComposeConfiguration) store(path string) error {
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
