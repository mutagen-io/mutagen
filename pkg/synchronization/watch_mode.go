package synchronization

import (
	"fmt"
)

// IsDefault indicates whether or not the watch mode is
// WatchMode_WatchModeDefault.
func (m WatchMode) IsDefault() bool {
	return m == WatchMode_WatchModeDefault
}

// MarshalText implements encoding.TextMarshaler.MarshalText.
func (m WatchMode) MarshalText() ([]byte, error) {
	var result string
	switch m {
	case WatchMode_WatchModeDefault:
	case WatchMode_WatchModePortable:
		result = "portable"
	case WatchMode_WatchModeForcePoll:
		result = "force-poll"
	case WatchMode_WatchModeNoWatch:
		result = "no-watch"
	default:
		result = "unknown"
	}
	return []byte(result), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText.
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
		return fmt.Errorf("unknown watch mode specification: %s", text)
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
