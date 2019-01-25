package sync

import (
	"github.com/pkg/errors"
)

// IsDefault indicates whether or not the synchronization mode is
// SynchronizationMode_SynchronizationModeDefault.
func (m SynchronizationMode) IsDefault() bool {
	return m == SynchronizationMode_SynchronizationModeDefault
}

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
func (m *SynchronizationMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a VCS mode.
	switch text {
	case "symmetric":
		*m = SynchronizationMode_SynchronizationModeSymmetric
	case "source-wins":
		*m = SynchronizationMode_SynchronizationModeSourceWins
	case "mirror-safe":
		*m = SynchronizationMode_SynchronizationModeMirrorSafe
	case "mirror-exact":
		*m = SynchronizationMode_SynchronizationModeMirrorExact
	default:
		return errors.Errorf("unknown synchronization mode specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular synchronization mode is a
// valid, non-default value.
func (m SynchronizationMode) Supported() bool {
	switch m {
	case SynchronizationMode_SynchronizationModeSymmetric:
		return true
	case SynchronizationMode_SynchronizationModeSourceWins:
		return true
	case SynchronizationMode_SynchronizationModeMirrorSafe:
		return true
	case SynchronizationMode_SynchronizationModeMirrorExact:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a synchronization mode.
func (m SynchronizationMode) Description() string {
	switch m {
	case SynchronizationMode_SynchronizationModeDefault:
		return "Default"
	case SynchronizationMode_SynchronizationModeSymmetric:
		return "Symmetric"
	case SynchronizationMode_SynchronizationModeSourceWins:
		return "Source Wins"
	case SynchronizationMode_SynchronizationModeMirrorSafe:
		return "Mirror Safe"
	case SynchronizationMode_SynchronizationModeMirrorExact:
		return "Mirror Exact"
	default:
		return "Unknown"
	}
}
