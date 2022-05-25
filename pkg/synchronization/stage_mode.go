package synchronization

import (
	"fmt"
)

// IsDefault indicates whether or not the staging mode is
// StageMode_StageModeDefault.
func (m StageMode) IsDefault() bool {
	return m == StageMode_StageModeDefault
}

// MarshalText implements encoding.TextMarshaler.MarshalText.
func (m StageMode) MarshalText() ([]byte, error) {
	var result string
	switch m {
	case StageMode_StageModeDefault:
	case StageMode_StageModeMutagen:
		result = "mutagen"
	case StageMode_StageModeNeighboring:
		result = "neighboring"
	case StageMode_StageModeInternal:
		result = "internal"
	default:
		result = "unknown"
	}
	return []byte(result), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText.
func (m *StageMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a staging mode.
	switch text {
	case "mutagen":
		*m = StageMode_StageModeMutagen
	case "neighboring":
		*m = StageMode_StageModeNeighboring
	case "internal":
		*m = StageMode_StageModeInternal
	default:
		return fmt.Errorf("unknown staging mode specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular staging mode is a valid,
// non-default value.
func (m StageMode) Supported() bool {
	switch m {
	case StageMode_StageModeMutagen:
		return true
	case StageMode_StageModeNeighboring:
		return true
	case StageMode_StageModeInternal:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a staging mode.
func (m StageMode) Description() string {
	switch m {
	case StageMode_StageModeDefault:
		return "Default"
	case StageMode_StageModeMutagen:
		return "Mutagen Data Directory"
	case StageMode_StageModeNeighboring:
		return "Neighboring"
	case StageMode_StageModeInternal:
		return "Internal"
	default:
		return "Unknown"
	}
}
