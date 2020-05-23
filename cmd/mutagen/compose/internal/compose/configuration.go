package compose

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/compose-spec/compose-go/template"

	"github.com/mutagen-io/mutagen/pkg/configuration/forwarding"
	"github.com/mutagen-io/mutagen/pkg/configuration/synchronization"
)

// ForwardingConfiguration encodes a full forwarding session specification.
type ForwardingConfiguration struct {
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

// SynchronizationConfiguration encodes a full synchronization session
// specification.
type SynchronizationConfiguration struct {
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

// MutagenConfiguration encodes the Mutagen configuration found in a Docker
// Compose configuration file under the x-mutagen extension.
type MutagenConfiguration struct {
	// Forwarding represents the forwarding sessions to be created. If a
	// "defaults" key is present, it is treated as a template upon which other
	// configurations are layered, thus keeping syntactic compatibility with the
	// global Mutagen configuration file.
	Forwarding map[string]ForwardingConfiguration `yaml:"forward"`
	// Synchronization represents the forwarding sessions to be created. If a
	// "defaults" key is present, it is treated as a template upon which other
	// configurations are layered, thus keeping syntactic compatibility with the
	// global Mutagen configuration file.
	Synchronization map[string]SynchronizationConfiguration `yaml:"sync"`
}

// Configuration encodes portions of a Docker Compose configuration file.
type Configuration struct {
	// Version is the configuration file schema version.
	Version string `yaml:"version"`
	// Services are the services defined in the configuration file.
	Services map[string]yaml.Node `yaml:"services"`
	// Volumes are the volumes defined in the configuration file.
	Volumes map[string]yaml.Node `yaml:"volumes"`
	// Networks are the networks defined in the configuration file.
	Networks map[string]yaml.Node `yaml:"networks"`
	// Mutagen is the Mutagen configuration defined in the configuration file.
	Mutagen MutagenConfiguration `yaml:"x-mutagen"`
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

// ReadConfiguration reads, interpolates, and decodes a Docker Compose
// configuration file from the specified stream. If the stream contains multiple
// YAML documents, then only the first will be read. Interpolation is performed
// using the specified variable mapping.
func ReadConfiguration(stream io.Reader, variables map[string]string) (*Configuration, error) {
	// Wrap the stream in a YAML decoder.
	decoder := yaml.NewDecoder(stream)

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
	result := &Configuration{
		Version:  intermediate.Version,
		Services: intermediate.Services,
		Volumes:  intermediate.Volumes,
		Networks: intermediate.Networks,
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
	if err := decoder.Decode(&result.Mutagen); err != nil {
		return nil, fmt.Errorf("strict parsing of x-mutagen YAML failed: %w", err)
	}

	// Success.
	return result, nil
}

// LoadConfiguration reads, interpolates, and decodes a Docker Compose
// configuration file from the specified file. If the file contains multiple
// YAML documents, then only the first will be read. Interpolation is performed
// using the specified variable mapping.
func LoadConfiguration(path string, variables map[string]string) (*Configuration, error) {
	// Open the file and defer its closure.
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open configuration file: %w", err)
	}
	defer file.Close()

	// Perform decoding and interpolation.
	return ReadConfiguration(file, variables)
}
