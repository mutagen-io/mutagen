package core

import (
	"fmt"
)

// IsDefault indicates whether or not the permissions mode is
// PermissionsMode_PermissionsModeDefault.
func (m PermissionsMode) IsDefault() bool {
	return m == PermissionsMode_PermissionsModeDefault
}

// MarshalText implements encoding.TextMarshaler.MarshalText.
func (m PermissionsMode) MarshalText() ([]byte, error) {
	var result string
	switch m {
	case PermissionsMode_PermissionsModeDefault:
	case PermissionsMode_PermissionsModePortable:
		result = "portable"
	case PermissionsMode_PermissionsModeManual:
		result = "manual"
	default:
		result = "unknown"
	}
	return []byte(result), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText.
func (m *PermissionsMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a permissions mode.
	switch text {
	case "portable":
		*m = PermissionsMode_PermissionsModePortable
	case "manual":
		*m = PermissionsMode_PermissionsModeManual
	default:
		return fmt.Errorf("unknown permissions mode specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular permissions mode is a valid,
// non-default value.
func (m PermissionsMode) Supported() bool {
	switch m {
	case PermissionsMode_PermissionsModePortable:
		return true
	case PermissionsMode_PermissionsModeManual:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a permissions mode.
func (m PermissionsMode) Description() string {
	switch m {
	case PermissionsMode_PermissionsModeDefault:
		return "Default"
	case PermissionsMode_PermissionsModePortable:
		return "Portable"
	case PermissionsMode_PermissionsModeManual:
		return "Manual"
	default:
		return "Unknown"
	}
}
