package sync

import (
	"github.com/pkg/errors"
)

// IsDefault indicates whether or not the VCS ignore mode is
// IgnoreVCSMode_IgnoreVCSModeDefault.
func (m IgnoreVCSMode) IsDefault() bool {
	return m == IgnoreVCSMode_IgnoreVCSModeDefault
}

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
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
		return errors.Errorf("unknown VCS ignore specification: %s", text)
	}

	// Success.
	return nil
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
