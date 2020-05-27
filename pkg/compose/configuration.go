package compose

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/compose-spec/compose-go/template"

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
// synchronization sessions found under an x-mutagen extension field.
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

// configuration encodes a subset of a Docker Compose configuration file.
type configuration struct {
	// version is the configuration file schema version.
	version string
	// services are the services defined in the configuration file.
	services map[string]struct{}
	// volumes are the volumes defined in the configuration file.
	volumes map[string]struct{}
	// networks are the networks defined in the configuration file.
	networks map[string]struct{}
	// mutagen is the Mutagen configuration defined in the configuration file.
	mutagen mutagenConfiguration
}

// intermediateConfiguration is an intermediate configuration structure used for
// non-strict YAML decoding. It allows configuration loading to separate any
// top-level x-mutagen YAML configuration for separate strict decoding.
type intermediateConfiguration struct {
	// Version is the configuration file schema version.
	Version string `yaml:"version"`
	// Services are the services defined in the configuration file.
	Services map[string]yaml.Node `yaml:"services"`
	// Volumes are the volumes defined in the configuration file.
	Volumes map[string]yaml.Node `yaml:"volumes"`
	// Networks are the networks defined in the configuration file.
	Networks map[string]yaml.Node `yaml:"networks"`
	// Mutagen is the Mutagen configuration defined in the configuration file.
	Mutagen yaml.Node `yaml:"x-mutagen"`
}

// interpolateNode performs recursive interpolation on a raw YAML hierarchy
// using the specified mapping. It only performs interpolation on scalar value
// nodes, not keys.
func interpolateNode(node *yaml.Node, mapping template.Mapping) error {
	// Handle interpolation based on the node type.
	switch node.Kind {
	case yaml.DocumentNode:
		fallthrough
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if err := interpolateNode(child, mapping); err != nil {
				return err
			}
		}
	case yaml.MappingNode:
		if len(node.Content)%2 != 0 {
			return errors.New("mapping node with unbalanced key/value count")
		}
		for i := 1; i < len(node.Content); i += 2 {
			if err := interpolateNode(node.Content[i], mapping); err != nil {
				return err
			}
		}
	case yaml.ScalarNode:
		var err error
		if node.Value, err = template.Substitute(node.Value, mapping); err != nil {
			return fmt.Errorf("unable to interpolate value (%s): %w", node.Value, err)
		}
	case yaml.AliasNode:
	default:
		return errors.New("unknown YAML node kind")
	}

	// Success.
	return nil
}

// yamlMapToStructMap is a conversion utility function that replaces the generic
// YAML nodes in intermediate representation nodes with empty structs. This is
// simply for the sake of keeping the API surface cleaner.
func yamlMapToStructMap(value map[string]yaml.Node) map[string]struct{} {
	result := make(map[string]struct{}, len(value))
	for key := range value {
		result[key] = struct{}{}
	}
	return result
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
		return nil, fmt.Errorf("unable to decode YAML: %w", err)
	}

	// Create the interpolation mapping.
	mapping := func(key string) (string, bool) {
		value, ok := variables[key]
		return value, ok
	}

	// Perform interpolation.
	if err := interpolateNode(&root, mapping); err != nil {
		return nil, fmt.Errorf("variable interpolation failed: %w", err)
	}

	// Decode the document into a more concrete structure that will allow us to
	// separate the x-mutagen specification for further validation. At this
	// point we still want non-strict decoding because we want to allow for
	// unknown top-level keys (since we need to play nice with top-level
	// extension fields).
	var intermediate intermediateConfiguration
	if err := root.Decode(&intermediate); err != nil {
		return nil, fmt.Errorf("unable to destructure configuration file: %w", err)
	}

	// Convert the configuration fields that don't require further processing.
	result := &configuration{
		version:  intermediate.Version,
		services: yamlMapToStructMap(intermediate.Services),
		volumes:  yamlMapToStructMap(intermediate.Volumes),
		networks: yamlMapToStructMap(intermediate.Networks),
	}

	// If there was no top-level x-mutagen specification, then we're done. For
	// some reason, decoding doesn't work if we make the Mutagen field a Node
	// pointer, it has to be a value. As such, the only way we can check for its
	// presence is to look at the node kind and look for a non-zero value.
	if intermediate.Mutagen.Kind == yaml.Kind(0) {
		return result, nil
	}

	// Extract and re-serialize the interpolated x-mutagen section. We have to
	// wrap the x-mutagen section in a document node for serialization to work.
	// TODO: Once go-yaml/yaml#460 is resolved, we won't need to perform this
	// re-serialization, we'll be able to perform a strict decoding into the
	// final structure directly from the decoded YAML node. This may be resolved
	// by go-yaml/yaml#557.
	mutagenYAML, err := yaml.Marshal(&yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{&intermediate.Mutagen},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to re-serialize x-mutagen YAML: %w", err)
	}

	// Now re-parse that YAML with strict decoding to ensure that it's correct.
	decoder = yaml.NewDecoder(bytes.NewReader(mutagenYAML))
	decoder.KnownFields(true)
	if err := decoder.Decode(&result.mutagen); err != nil {
		return nil, fmt.Errorf("strict parsing of x-mutagen YAML failed: %w", err)
	}

	// Success.
	return result, nil
}

// mutagenComposeConfiguration represents a Docker Compose configuration file
// for the Mutagen service.
type mutagenComposeConfiguration struct {
	// Version is the Docker Compose configuration file version.
	Version string `yaml:"version"`
	// Services are the Docker Compose services
	Services struct {
		// Mutagen is the Mutagen service.
		// TODO: The key for this field really ought to come from the
		// mutagenServiceName constant. We should at least add a test to enforce
		// that they match.
		Mutagen struct {
			// Build is the build context for the Mutagen service.
			Build string `yaml:"build"`
			// Init indicates whether or not a Docker init process should be
			// used to wrap the Mutagen container entry point.
			Init bool `yaml:"init,omitempty"`
			// Networks are the network dependencies for the Mutagen service.
			Networks []string `yaml:"networks"`
			// Volumes are the volume dependencies for the Mutagen service.
			Volumes []string `yaml:"volumes"`
		} `yaml:"mutagen"`
	} `yaml:"services"`
}

// store encodes the configuration to YAML and writes it to the specified path.
func (c *mutagenComposeConfiguration) store(path string) error {
	// Open the output file and defer its closure.
	output, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
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
