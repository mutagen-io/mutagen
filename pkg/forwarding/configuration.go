package forwarding

import (
	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

// YAMLConfiguration represents a YAML-based Mutagen forwarding session
// configuration.
type YAMLConfiguration struct {
	// Socket contains parameters related to Unix domain socket handling.
	Socket struct {
		// OverwriteMode specifies the default socket overwrite mode to use for
		// Unix domain socket endpoints.
		OverwriteMode SocketOverwriteMode `yaml:"overwriteMode"`
		// Owner specifies the owner identifier to use for Unix domain listener
		// sockets.
		Owner string `yaml:"owner"`
		// Group specifies the group identifier to use for Unix domain listener
		// sockets.
		Group string `yaml:"group"`
		// PermissionMode specifies the permission mode to use for Unix domain
		// listener sockets.
		PermissionMode filesystem.Mode `yaml:"permissionMode"`
	} `yaml:"socket"`
}

// Configuration converts a YAML session configuration to a Protocol Buffers
// session configuration. It does not validate the resulting configuration.
func (c *YAMLConfiguration) Configuration() *Configuration {
	return &Configuration{
		SocketOverwriteMode:  c.Socket.OverwriteMode,
		SocketOwner:          c.Socket.Owner,
		SocketGroup:          c.Socket.Group,
		SocketPermissionMode: uint32(c.Socket.PermissionMode),
	}
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
