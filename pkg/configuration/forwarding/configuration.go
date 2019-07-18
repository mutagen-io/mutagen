package forwarding

import (
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
)

// Configuration represents a YAML-based Mutagen forwarding session
// configuration.
type Configuration struct {
	// Socket contains parameters related to Unix domain socket handling.
	Socket struct {
		// OverwriteMode specifies the default socket overwrite mode to use for
		// Unix domain socket endpoints.
		OverwriteMode forwarding.SocketOverwriteMode `yaml:"overwriteMode"`
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
func (c *Configuration) Configuration() *forwarding.Configuration {
	return &forwarding.Configuration{
		SocketOverwriteMode:  c.Socket.OverwriteMode,
		SocketOwner:          c.Socket.Owner,
		SocketGroup:          c.Socket.Group,
		SocketPermissionMode: uint32(c.Socket.PermissionMode),
	}
}
