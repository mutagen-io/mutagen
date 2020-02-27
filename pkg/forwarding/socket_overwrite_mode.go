package forwarding

import (
	"github.com/pkg/errors"
)

// IsDefault indicates whether or not the socket overwrite mode is
// SocketOverwriteMode_SocketOverwriteModeDefault.
func (m SocketOverwriteMode) IsDefault() bool {
	return m == SocketOverwriteMode_SocketOverwriteModeDefault
}

// AttemptOverwrite indicates whether or not the socket overwrite mode is
// SocketOverwriteMode_SocketOverwriteModeOverwrite.
func (m SocketOverwriteMode) AttemptOverwrite() bool {
	return m == SocketOverwriteMode_SocketOverwriteModeOverwrite
}

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
func (m *SocketOverwriteMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a socket overwrite mode.
	switch text {
	case "leave":
		*m = SocketOverwriteMode_SocketOverwriteModeLeave
	case "overwrite":
		*m = SocketOverwriteMode_SocketOverwriteModeOverwrite
	default:
		return errors.Errorf("unknown socket overwrite mode specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular socket overwrite mode is a
// valid, non-default value.
func (m SocketOverwriteMode) Supported() bool {
	switch m {
	case SocketOverwriteMode_SocketOverwriteModeLeave:
		return true
	case SocketOverwriteMode_SocketOverwriteModeOverwrite:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a socket overwrite mode.
func (m SocketOverwriteMode) Description() string {
	switch m {
	case SocketOverwriteMode_SocketOverwriteModeDefault:
		return "Default"
	case SocketOverwriteMode_SocketOverwriteModeLeave:
		return "Leave"
	case SocketOverwriteMode_SocketOverwriteModeOverwrite:
		return "Overwrite"
	default:
		return "Unknown"
	}
}
