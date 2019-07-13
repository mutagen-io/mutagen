package forwarding

import (
	"github.com/pkg/errors"
)

// YAMLConfiguration represents a YAML-based Mutagen forwarding session
// configuration.
type YAMLConfiguration struct{}

// Configuration converts a YAML session configuration to a Protocol Buffers
// session configuration. It does not validate the resulting configuration.
func (c *YAMLConfiguration) Configuration() *Configuration {
	return &Configuration{}
}

// EnsureValid ensures that Configuration's invariants are respected. The
// validation of the configuration depends on whether or not it is
// endpoint-specific.
func (c *Configuration) EnsureValid(endpointSpecific bool) error {
	// A nil configuration is not considered valid.
	if c == nil {
		return errors.New("nil configuration")
	}

	// Success.
	return nil
}

// MergeConfigurations merges two configurations of differing priorities. Both
// configurations must be non-nil.
func MergeConfigurations(lower, higher *Configuration) *Configuration {
	// Create the resulting configuration.
	result := &Configuration{}

	// Done.
	return result
}
