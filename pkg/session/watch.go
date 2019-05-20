package session

import (
	"github.com/pkg/errors"
)

// IsDefault indicates whether or not the watch mode is
// WatchMode_WatchModeDefault.
func (m WatchMode) IsDefault() bool {
	return m == WatchMode_WatchModeDefault
}

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
func (m *WatchMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a watch mode.
	switch text {
	case "portable":
		*m = WatchMode_WatchModePortable
	case "force-poll":
		*m = WatchMode_WatchModeForcePoll
	case "no-watch":
		*m = WatchMode_WatchModeNoWatch
	default:
		return errors.Errorf("unknown watch mode specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular watch mode is a valid,
// non-default value.
func (m WatchMode) Supported() bool {
	switch m {
	case WatchMode_WatchModePortable:
		return true
	case WatchMode_WatchModeForcePoll:
		return true
	case WatchMode_WatchModeNoWatch:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a watch mode.
func (m WatchMode) Description() string {
	switch m {
	case WatchMode_WatchModeDefault:
		return "Default"
	case WatchMode_WatchModePortable:
		return "Portable"
	case WatchMode_WatchModeForcePoll:
		return "Force Poll"
	case WatchMode_WatchModeNoWatch:
		return "No Watch"
	default:
		return "Unknown"
	}
}
