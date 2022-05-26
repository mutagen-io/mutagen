package forwarding

import (
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
)

// Configuration represents forwarding session configuration.
type Configuration struct {
	// Socket contains parameters related to Unix domain socket handling.
	Socket struct {
		// OverwriteMode specifies the default socket overwrite mode to use for
		// Unix domain socket endpoints.
		OverwriteMode forwarding.SocketOverwriteMode `json:"overwriteMode,omitempty" yaml:"overwriteMode" mapstructure:"overwriteMode"`
		// Owner specifies the owner identifier to use for Unix domain listener
		// sockets.
		Owner string `json:"owner,omitempty" yaml:"owner" mapstructure:"owner"`
		// Group specifies the group identifier to use for Unix domain listener
		// sockets.
		Group string `json:"group,omitempty" yaml:"group" mapstructure:"group"`
		// PermissionMode specifies the permission mode to use for Unix domain
		// listener sockets.
		PermissionMode filesystem.Mode `json:"permissionMode,omitempty" yaml:"permissionMode" mapstructure:"permissionMode"`
	} `json:"socket" yaml:"socket" mapstructure:"socket"`
}

// loadFromInternal sets a configuration to match an internal Protocol Buffers
// representation. The configuration must be valid.
func (c *Configuration) loadFromInternal(configuration *forwarding.Configuration) {
	// Propagate socket configuration.
	c.Socket.OverwriteMode = configuration.SocketOverwriteMode
	c.Socket.Owner = configuration.SocketOwner
	c.Socket.Group = configuration.SocketGroup
	c.Socket.PermissionMode = filesystem.Mode(configuration.SocketPermissionMode)
}

// ToInternal converts a public configuration representation to an internal
// Protocol Buffers session configuration. It does not validate the resulting
// configuration.
func (c *Configuration) ToInternal() *forwarding.Configuration {
	return &forwarding.Configuration{
		SocketOverwriteMode:  c.Socket.OverwriteMode,
		SocketOwner:          c.Socket.Owner,
		SocketGroup:          c.Socket.Group,
		SocketPermissionMode: uint32(c.Socket.PermissionMode),
	}
}
