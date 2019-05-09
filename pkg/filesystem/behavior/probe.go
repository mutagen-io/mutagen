package behavior

import (
	"github.com/pkg/errors"
)

// IsDefault indicates whether or not the probe mode is
// ProbeMode_ProbeModeDefault.
func (m ProbeMode) IsDefault() bool {
	return m == ProbeMode_ProbeModeDefault
}

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
func (m *ProbeMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a probe mode.
	switch text {
	case "probe":
		*m = ProbeMode_ProbeModeProbe
	case "assume":
		*m = ProbeMode_ProbeModeAssume
	default:
		return errors.Errorf("unknown probe mode specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular probe mode is a valid,
// non-default value.
func (m ProbeMode) Supported() bool {
	switch m {
	case ProbeMode_ProbeModeProbe:
		return true
	case ProbeMode_ProbeModeAssume:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a probe mode.
func (m ProbeMode) Description() string {
	switch m {
	case ProbeMode_ProbeModeDefault:
		return "Default"
	case ProbeMode_ProbeModeProbe:
		return "Probe"
	case ProbeMode_ProbeModeAssume:
		return "Assume"
	default:
		return "Unknown"
	}
}
