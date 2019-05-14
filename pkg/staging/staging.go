package staging

import (
	"github.com/pkg/errors"
)

// IsDefault indicates whether or not the staging mode is
// StagingMode_StagingModeDefault.
func (m StagingMode) IsDefault() bool {
	return m == StagingMode_StagingModeDefault
}

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
func (m *StagingMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a staging mode.
	switch text {
	case "mutagen":
		*m = StagingMode_StagingModeMutagen
	case "neighboring":
		*m = StagingMode_StagingModeNeighboring
	default:
		return errors.Errorf("unknown staging mode specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular staging mode is a valid,
// non-default value.
func (m StagingMode) Supported() bool {
	switch m {
	case StagingMode_StagingModeMutagen:
		return true
	case StagingMode_StagingModeNeighboring:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a staging mode.
func (m StagingMode) Description() string {
	switch m {
	case StagingMode_StagingModeDefault:
		return "Default"
	case StagingMode_StagingModeMutagen:
		return "Mutagen"
	case StagingMode_StagingModeNeighboring:
		return "Neighboring"
	default:
		return "Unknown"
	}
}
