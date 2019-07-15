package forwarding

import (
	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

// Supported indicates whether or not the session version is supported.
func (v Version) Supported() bool {
	switch v {
	case Version_Version1:
		return true
	default:
		return false
	}
}

// DefaultSocketOverwriteMode returns the default socket overwrite mode for the
// session version.
func (v Version) DefaultSocketOverwriteMode() SocketOverwriteMode {
	switch v {
	case Version_Version1:
		return SocketOverwriteMode_SocketOverwriteModeLeave
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultSocketPermissionMode returns the default socket permission mode for
// the session version.
func (v Version) DefaultSocketPermissionMode() filesystem.Mode {
	switch v {
	case Version_Version1:
		return filesystem.ModePermissionUserRead | filesystem.ModePermissionUserWrite
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultSocketOwnerSpecification returns the default socket owner
// specification for the session version.
func (v Version) DefaultSocketOwnerSpecification() string {
	switch v {
	case Version_Version1:
		return ""
	default:
		panic("unknown or unsupported session version")
	}
}

// DefaultSocketGroupSpecification returns the default socket group
// specification for the session version.
func (v Version) DefaultSocketGroupSpecification() string {
	switch v {
	case Version_Version1:
		return ""
	default:
		panic("unknown or unsupported session version")
	}
}
