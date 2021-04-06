package core

import (
	"fmt"
)

// IsDefault indicates whether or not the symbolic link mode is
// SymbolicLinkMode_SymbolicLinkModeDefault.
func (m SymbolicLinkMode) IsDefault() bool {
	return m == SymbolicLinkMode_SymbolicLinkModeDefault
}

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
func (m *SymbolicLinkMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a VCS mode.
	switch text {
	case "ignore":
		*m = SymbolicLinkMode_SymbolicLinkModeIgnore
	case "portable":
		*m = SymbolicLinkMode_SymbolicLinkModePortable
	case "posix-raw":
		*m = SymbolicLinkMode_SymbolicLinkModePOSIXRaw
	default:
		return fmt.Errorf("unknown symbolic link mode specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular symbolic link mode is a
// valid, non-default value.
func (m SymbolicLinkMode) Supported() bool {
	switch m {
	case SymbolicLinkMode_SymbolicLinkModeIgnore:
		return true
	case SymbolicLinkMode_SymbolicLinkModePortable:
		return true
	case SymbolicLinkMode_SymbolicLinkModePOSIXRaw:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a symbolic link mode.
func (m SymbolicLinkMode) Description() string {
	switch m {
	case SymbolicLinkMode_SymbolicLinkModeDefault:
		return "Default"
	case SymbolicLinkMode_SymbolicLinkModeIgnore:
		return "Ignore"
	case SymbolicLinkMode_SymbolicLinkModePortable:
		return "Portable"
	case SymbolicLinkMode_SymbolicLinkModePOSIXRaw:
		return "POSIX Raw"
	default:
		return "Unknown"
	}
}
