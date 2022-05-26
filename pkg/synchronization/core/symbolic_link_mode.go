package core

import (
	"fmt"
)

// IsDefault indicates whether or not the symbolic link mode is
// SymbolicLinkMode_SymbolicLinkModeDefault.
func (m SymbolicLinkMode) IsDefault() bool {
	return m == SymbolicLinkMode_SymbolicLinkModeDefault
}

// MarshalText implements encoding.TextMarshaler.MarshalText.
func (m SymbolicLinkMode) MarshalText() ([]byte, error) {
	var result string
	switch m {
	case SymbolicLinkMode_SymbolicLinkModeDefault:
	case SymbolicLinkMode_SymbolicLinkModeIgnore:
		result = "ignore"
	case SymbolicLinkMode_SymbolicLinkModePortable:
		result = "portable"
	case SymbolicLinkMode_SymbolicLinkModePOSIXRaw:
		result = "posix-raw"
	default:
		result = "unknown"
	}
	return []byte(result), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText.
func (m *SymbolicLinkMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a symbolic link mode.
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
