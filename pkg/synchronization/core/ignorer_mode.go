package core

import (
	"fmt"
)

// IsDefault indicates whether or not the ignorer mode is
// IgnorerMode_IgnorerModeDefault.
func (m IgnorerMode) IsDefault() bool {
	return m == IgnorerMode_IgnorerModeDefault
}

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
func (m *IgnorerMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a ignorer mode.
	switch text {
	case "dockerignore":
		*m = IgnorerMode_IgnorerModeDocker
	case "default", "gitignore":
		*m = IgnorerMode_IgnorerModeDefault
	default:
		return fmt.Errorf("unknown ignorer mode specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular ignorer mode is a
// valid, non-default value.
func (m IgnorerMode) Supported() bool {
	switch m {
	case IgnorerMode_IgnorerModeDefault:
		return true
	case IgnorerMode_IgnorerModeDocker:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a ignorer mode.
func (m IgnorerMode) Description() string {
	switch m {
	case IgnorerMode_IgnorerModeDefault:
		return "Default"
	case IgnorerMode_IgnorerModeDocker:
		return "Docker Ignore"
	default:
		return "Unknown"
	}
}
