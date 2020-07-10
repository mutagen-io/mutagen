package configuration

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/mutagen-io/mutagen/pkg/configuration/forwarding"
	"github.com/mutagen-io/mutagen/pkg/configuration/synchronization"
)

// ForwardingConfiguration encodes a forwarding session specification.
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

// SynchronizationConfiguration encodes a synchronization session specification.
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

// MutagenConfiguration encodes collections of Mutagen forwarding and
// synchronization sessions found under an "x-mutagen" extension field.
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

// intermediate is an intermediate configuration representation used for loading
// Docker Compose configuration files. It is necessary to allow for strict
// decoding of the "x-mutagen" section while using flexible decoding for all
// other sections of the file. It is also necessary in order to avoid exposing
// yaml.Node types in this package's API. We have to use yaml.Node types for our
// map value types because the yaml package won't decode key-only map entries
// into a map[string]struct{} type (it will just ignore them). Since key-only
// map entries are common in Docker Compose volume and network definitions, we
// use yaml.Node value types to allow for indication of this absence while still
// capturing volume and network names.
type intermediate struct {
	// Version is the configuration file schema version.
	Version string `yaml:"version"`
	// Services are the services defined in the file. See the struct comments to
	// understand why we use a yaml.Node type for the map value. This is less
	// essential for the "services" section since service definitions should
	// have non-empty contents, but using yaml.Node is more consistent with the
	// Volumes and Networks fields below and enables future decoding.
	Services map[string]yaml.Node `yaml:"services"`
	// Volumes are the volumes defined in the file. See the struct comments to
	// understand why we use a yaml.Node type for the map value.
	Volumes map[string]yaml.Node `yaml:"volumes"`
	// Networks are the networks defined in the file. See the struct comments to
	// understand why we use a yaml.Node type for the map value.
	Networks map[string]yaml.Node `yaml:"networks"`
	// XMutagen is the raw Mutagen configuration defined in the file.
	XMutagen yaml.Node `yaml:"x-mutagen"`
}

// Configuration represents a single Docker Compose configuration file with
// Mutagen sessions specified using an "x-mutagen" extension.
type Configuration struct {
	// Version is the configuration file schema version.
	Version string
	// Services are the services defined in the file.
	Services map[string]struct{}
	// Volumes are the volumes defined in the file.
	Volumes map[string]struct{}
	// Networks are the networks defined in the file.
	Networks map[string]struct{}
	// Mutagen is the Mutagen session configuration found in the file. The
	// session specifications in this field are not validated by Load.
	Mutagen MutagenConfiguration
}

// yamlMapToEmptyStructMap converts a map of string-to-yaml.Node to a map of
// string-to-empty-struct. It preserves the distinction between nil and empty.
func yamlMapToEmptyStructMap(source map[string]yaml.Node) map[string]struct{} {
	// Handle the nil case.
	if source == nil {
		return nil
	}

	// Handle the non-nil (but potentially empty) case.
	result := make(map[string]struct{}, len(source))
	for key := range source {
		result[key] = struct{}{}
	}

	// Done.
	return result
}

// Load reads, interpolates, and decodes a Docker Compose YAML configuration
// from the specified file. If the file contains multiple YAML documents, then
// only the first will be read. Interpolation is performed using the specified
// variable mapping. The only validation performed by this function is on the
// YAML syntax and the keys and value types provided as part of the "x-mutagen"
// configuration. No validation is performed on Docker Compose fields or Mutagen
// session specifications.
func Load(path string, variables map[string]string) (*Configuration, error) {
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
	if err := interpolate(&root, mapping); err != nil {
		return nil, fmt.Errorf("variable interpolation failed: %w", err)
	}

	// Decode the document into a more concrete configuration structure so that
	// we can extract and validate the Mutagen configuration.
	var decoded intermediate
	if err := root.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("unable to parse configuration file: %w", err)
	}

	// Create the resulting configuration, excluding the Mutagen configuration.
	// We convert our maps to avoid exposing yaml.Node types in the package API.
	// See the comments for intermediate to understand why we use yaml.Node.
	result := &Configuration{
		Version:  decoded.Version,
		Services: yamlMapToEmptyStructMap(decoded.Services),
		Volumes:  yamlMapToEmptyStructMap(decoded.Volumes),
		Networks: yamlMapToEmptyStructMap(decoded.Networks),
	}

	// If there was no top-level "x-mutagen" specification, then we're done.
	if decoded.XMutagen.IsZero() {
		return result, nil
	}

	// Otherwise, extract and re-serialize the interpolated "x-mutagen" section
	// and re-parse it using strict decoding (i.e. only allowing known fields).
	//
	// TODO: Once go-yaml/yaml#460 is resolved, we won't need to perform this
	// re-serialization since we'll be able to perform a strict decoding into
	// the final structure directly from the decoded (and interpolated) YAML
	// node. This is being implemented in go-yaml/yaml#557.
	xMutagenYAML, err := yaml.Marshal(&yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{&decoded.XMutagen},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to re-serialize Mutagen YAML: %w", err)
	}
	decoder = yaml.NewDecoder(bytes.NewReader(xMutagenYAML))
	decoder.KnownFields(true)
	if err := decoder.Decode(&result.Mutagen); err != nil {
		return nil, fmt.Errorf("strict parsing of Mutagen YAML failed: %w", err)
	}

	// Success.
	return result, nil
}
