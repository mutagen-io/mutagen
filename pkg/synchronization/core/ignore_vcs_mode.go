package core

import (
	"errors"
	"fmt"
)

// IsDefault indicates whether or not the VCS ignore mode is
// IgnoreVCSMode_IgnoreVCSModeDefault.
func (m IgnoreVCSMode) IsDefault() bool {
	return m == IgnoreVCSMode_IgnoreVCSModeDefault
}

// MarshalJSON implements encoding/json.Marshaler.MarshalJSON.
func (m IgnoreVCSMode) MarshalJSON() ([]byte, error) {
	var result string
	switch m {
	case IgnoreVCSMode_IgnoreVCSModeDefault:
		return nil, errors.New("default VCS ignore mode has no JSON representation")
	case IgnoreVCSMode_IgnoreVCSModeIgnore:
		result = "true"
	case IgnoreVCSMode_IgnoreVCSModePropagate:
		result = "false"
	default:
		return nil, fmt.Errorf("invalid VCS ignore mode: %d", m)
	}
	return []byte(result), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText.
func (m *IgnoreVCSMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a VCS mode.
	switch text {
	case "true":
		*m = IgnoreVCSMode_IgnoreVCSModeIgnore
	case "false":
		*m = IgnoreVCSMode_IgnoreVCSModePropagate
	default:
		return fmt.Errorf("unknown VCS ignore specification: %s", text)
	}

	// Success.
	return nil
}

// UnmarshalJSON implements encoding/json.Unmarshaler.UnmarshalJSON.
func (m *IgnoreVCSMode) UnmarshalJSON(textBytes []byte) error {
	return m.UnmarshalText(textBytes)
}

// Supported indicates whether or not a particular VCS ignore mode is a valid,
// non-default value.
func (m IgnoreVCSMode) Supported() bool {
	switch m {
	case IgnoreVCSMode_IgnoreVCSModeIgnore:
		return true
	case IgnoreVCSMode_IgnoreVCSModePropagate:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a VCS ignore mode.
func (m IgnoreVCSMode) Description() string {
	switch m {
	case IgnoreVCSMode_IgnoreVCSModeDefault:
		return "Default"
	case IgnoreVCSMode_IgnoreVCSModeIgnore:
		return "Ignore"
	case IgnoreVCSMode_IgnoreVCSModePropagate:
		return "Propagate"
	default:
		return "Unknown"
	}
}
