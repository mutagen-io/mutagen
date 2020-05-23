package compose

import (
	"gopkg.in/yaml.v3"

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
	Services map[string]*yaml.Node `yaml:"services"`
	// Volumes are the volumes defined in the configuration file.
	Volumes map[string]*yaml.Node `yaml:"volumes"`
	// Networks are the networks defined in the configuration file.
	Networks map[string]*yaml.Node `yaml:"networks"`
	// Configs are the configs defined in the configuration file.
	Configs map[string]*yaml.Node `yaml:"configs"`
	// Secrets are the secrets defined in the configuration file.
	Secrets map[string]*yaml.Node `yaml:"secrets"`
	// Mutagen is the Mutagen configuration defined in the configuration file.
	Mutagen MutagenConfiguration `yaml:"x-mutagen"`
}
