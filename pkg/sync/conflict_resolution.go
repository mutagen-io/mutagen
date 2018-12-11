package sync

import (
	"github.com/pkg/errors"
)

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
func (m *ConflictResolutionMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a VCS mode.
	switch text {
	case "safe":
		*m = ConflictResolutionMode_ConflictResolutionModeSafe
	case "alpha-wins":
		*m = ConflictResolutionMode_ConflictResolutionModeAlphaWins
	case "beta-wins":
		*m = ConflictResolutionMode_ConflictResolutionModeBetaWins
	case "alpha-wins-all":
		*m = ConflictResolutionMode_ConflictResolutionModeAlphaWinsAll
	case "beta-wins-all":
		*m = ConflictResolutionMode_ConflictResolutionModeBetaWinsAll
	default:
		return errors.Errorf("unknown conflict resolution mode specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular conflict resolution mode is a
// valid, non-default value.
func (m ConflictResolutionMode) Supported() bool {
	switch m {
	case ConflictResolutionMode_ConflictResolutionModeSafe:
		return true
	case ConflictResolutionMode_ConflictResolutionModeAlphaWins:
		return true
	case ConflictResolutionMode_ConflictResolutionModeBetaWins:
		return true
	case ConflictResolutionMode_ConflictResolutionModeAlphaWinsAll:
		return true
	case ConflictResolutionMode_ConflictResolutionModeBetaWinsAll:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a conflict resolution
// mode.
func (m ConflictResolutionMode) Description() string {
	switch m {
	case ConflictResolutionMode_ConflictResolutionModeDefault:
		return "Default"
	case ConflictResolutionMode_ConflictResolutionModeSafe:
		return "Safe"
	case ConflictResolutionMode_ConflictResolutionModeAlphaWins:
		return "Alpha Wins"
	case ConflictResolutionMode_ConflictResolutionModeBetaWins:
		return "Beta Wins"
	case ConflictResolutionMode_ConflictResolutionModeAlphaWinsAll:
		return "Alpha Wins All"
	case ConflictResolutionMode_ConflictResolutionModeBetaWinsAll:
		return "Beta Wins All"
	default:
		return "Unknown"
	}
}
