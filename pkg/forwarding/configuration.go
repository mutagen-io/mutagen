package forwarding

import (
	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

// Equal returns whether or not the configuration is equivalent to another. The
// result of this method is only valid if both configurations are valid.
func (c *Configuration) Equal(other *Configuration) bool {
	// Ensure that both are non-nil.
	if c == nil || other == nil {
		return false
	}

	// Perform an equivalence check.
	return c.SocketOverwriteMode == other.SocketOverwriteMode &&
		c.SocketOwner == other.SocketOwner &&
		c.SocketGroup == other.SocketGroup &&
		c.SocketPermissionMode == other.SocketPermissionMode
}

// EnsureValid ensures that Configuration's invariants are respected. The
// validation of the configuration depends on whether or not it is
// endpoint-specific.
func (c *Configuration) EnsureValid(endpointSpecific bool) error {
	// A nil configuration is not considered valid.
	if c == nil {
		return errors.New("nil configuration")
	}

	// Verify that the socket overwrite mode is unspecified or supported for
	// usage.
	if !(c.SocketOverwriteMode.IsDefault() || c.SocketOverwriteMode.Supported()) {
		return errors.New("unknown or unsupported socket overwrite mode")
	}

	// Verify the socket owner specification.
	if c.SocketOwner != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(c.SocketOwner); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid socket owner specification")
		}
	}

	// Verify the socket group specification.
	if c.SocketGroup != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(c.SocketGroup); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid socket group specification")
		}
	}

	// We don't verify the socket permission mode because there's not really any
	// way to know if it's a sane value.

	// Success.
	return nil
}

// MergeConfigurations merges two configurations of differing priorities. Both
// configurations must be non-nil.
func MergeConfigurations(lower, higher *Configuration) *Configuration {
	// Create the resulting configuration.
	result := &Configuration{}

	// Merge socket overwrite mode.
	if !higher.SocketOverwriteMode.IsDefault() {
		result.SocketOverwriteMode = higher.SocketOverwriteMode
	} else {
		result.SocketOverwriteMode = lower.SocketOverwriteMode
	}

	// Merge socket owner.
	if higher.SocketOwner != "" {
		result.SocketOwner = higher.SocketOwner
	} else {
		result.SocketOwner = lower.SocketOwner
	}

	// Merge socket group.
	if higher.SocketGroup != "" {
		result.SocketGroup = higher.SocketGroup
	} else {
		result.SocketGroup = lower.SocketGroup
	}

	// Merge socket permission mode.
	if higher.SocketPermissionMode != 0 {
		result.SocketPermissionMode = higher.SocketPermissionMode
	} else {
		result.SocketPermissionMode = lower.SocketPermissionMode
	}

	// Done.
	return result
}
