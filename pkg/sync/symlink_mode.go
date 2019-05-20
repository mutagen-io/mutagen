package sync

import (
	"github.com/pkg/errors"
)

// IsDefault indicates whether or not the symbolic link handling mode is
// SymlinkMode_SymlinkModeDefault.
func (m SymlinkMode) IsDefault() bool {
	return m == SymlinkMode_SymlinkModeDefault
}

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
func (m *SymlinkMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a VCS mode.
	switch text {
	case "ignore":
		*m = SymlinkMode_SymlinkModeIgnore
	case "portable":
		*m = SymlinkMode_SymlinkModePortable
	case "posix-raw":
		*m = SymlinkMode_SymlinkModePOSIXRaw
	default:
		return errors.Errorf("unknown symlink mode specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular symlink mode is a valid,
// non-default value.
func (m SymlinkMode) Supported() bool {
	switch m {
	case SymlinkMode_SymlinkModeIgnore:
		return true
	case SymlinkMode_SymlinkModePortable:
		return true
	case SymlinkMode_SymlinkModePOSIXRaw:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a symlink mode.
func (m SymlinkMode) Description() string {
	switch m {
	case SymlinkMode_SymlinkModeDefault:
		return "Default"
	case SymlinkMode_SymlinkModeIgnore:
		return "Ignore"
	case SymlinkMode_SymlinkModePortable:
		return "Portable"
	case SymlinkMode_SymlinkModePOSIXRaw:
		return "POSIX Raw"
	default:
		return "Unknown"
	}
}
