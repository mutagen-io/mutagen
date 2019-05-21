package session

import (
	"github.com/pkg/errors"
)

// IsDefault indicates whether or not the staging mode is
// StageMode_StageModeDefault.
func (m StageMode) IsDefault() bool {
	return m == StageMode_StageModeDefault
}

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
func (m *StageMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a staging mode.
	switch text {
	case "mutagen":
		*m = StageMode_StageModeMutagen
	case "neighboring":
		*m = StageMode_StageModeNeighboring
	default:
		return errors.Errorf("unknown staging mode specification: %s", text)
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
	default:
		return "Unknown"
	}
}
