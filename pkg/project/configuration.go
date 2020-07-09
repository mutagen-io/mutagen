package project

import (
	"github.com/mutagen-io/mutagen/pkg/configuration/forwarding"
	"github.com/mutagen-io/mutagen/pkg/configuration/synchronization"
	"github.com/mutagen-io/mutagen/pkg/encoding"
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

// FlushOnCreateBehavior is a custom YAML type that can encode various
// flush-on-create specifications, including a lack of specification.
type FlushOnCreateBehavior uint8

const (
	// FlushOnCreateBehaviorDefault indicates that flush-on-create behavior is
	// unspecified.
	FlushOnCreateBehaviorDefault FlushOnCreateBehavior = iota
	// FlushOnCreateBehaviorNoFlush indicates that flush-on-create behavior has
	// been disabled.
	FlushOnCreateBehaviorNoFlush
	// FlushOnCreateBehaviorNoFlush indicates that flush-on-create behavior has
	// been enabled.
	FlushOnCreateBehaviorFlush
)

// IsDefault indicates whether or not the flush-on-create behavior is
// FlushOnCreateBehaviorDefault.
func (b FlushOnCreateBehavior) IsDefault() bool {
	return b == FlushOnCreateBehaviorDefault
}

// FlushOnCreate converts the behavior specification to an actual boolean
// indicating behavior.
func (b FlushOnCreateBehavior) FlushOnCreate() bool {
	switch b {
	case FlushOnCreateBehaviorDefault:
		return false
	case FlushOnCreateBehaviorNoFlush:
		return false
	case FlushOnCreateBehaviorFlush:
		return true
	default:
		panic("unhandled flush-on-create behavior")
	}
}

// UnmarshalYAML implements Unmarshaler.UnmarshalYAML.
func (b *FlushOnCreateBehavior) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Call the underlying unmarshaling function.
	var flush bool
	if err := unmarshal(&flush); err != nil {
		return err
	}

	// Set behavior.
	if flush {
		*b = FlushOnCreateBehaviorFlush
	} else {
		*b = FlushOnCreateBehaviorNoFlush
	}

	// Success.
	return nil
}

// SynchronizationConfiguration encodes a synchronization session specification.
type SynchronizationConfiguration struct {
	// Alpha is the alpha URL for the session.
	Alpha string `yaml:"alpha"`
	// Beta is the beta URL for the session.
	Beta string `yaml:"beta"`
	// FlushOnCreate indicates the flush-on-create behavior for the session.
	FlushOnCreate FlushOnCreateBehavior `yaml:"flushOnCreate"`
	// Configuration is the configuration for the session.
	Configuration synchronization.Configuration `yaml:",inline"`
	// ConfigurationAlpha is the alpha-specific configuration for the session.
	ConfigurationAlpha synchronization.Configuration `yaml:"configurationAlpha"`
	// ConfigurationBeta is the beta-specific configuration for the session.
	ConfigurationBeta synchronization.Configuration `yaml:"configurationBeta"`
}

// Configuration is the orchestration configuration object type.
type Configuration struct {
	// BeforeCreate are setup commands to be run before session creation.
	BeforeCreate []string `yaml:"beforeCreate"`
	// AfterCreate are setup commands to be run after session creation.
	AfterCreate []string `yaml:"afterCreate"`
	// BeforePause are setup commands to be run before session pausing.
	BeforePause []string `yaml:"beforePause"`
	// AfterPause are setup commands to be run after session pausing.
	AfterPause []string `yaml:"afterPause"`
	// BeforeResume are setup commands to be run before session resumption.
	BeforeResume []string `yaml:"beforeResume"`
	// AfterResume are setup commands to be run after session resumption.
	AfterResume []string `yaml:"afterResume"`
	// BeforeTerminate are teardown commands to be run before session
	// termination.
	BeforeTerminate []string `yaml:"beforeTerminate"`
	// AfterTerminate are teardown commands to be run after session termination.
	AfterTerminate []string `yaml:"afterTerminate"`
	// Commands are commands that can be invoked while a project is running.
	Commands map[string]string `yaml:"commands"`
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

// LoadConfiguration attempts to load a YAML-based Mutagen orchestration
// configuration file from the specified path.
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
