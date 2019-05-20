package session

import (
	"github.com/pkg/errors"
)

// IsDefault indicates whether or not the scan mode is ScanMode_ScanModeDefault.
func (m ScanMode) IsDefault() bool {
	return m == ScanMode_ScanModeDefault
}

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
func (m *ScanMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a scan mode.
	switch text {
	case "full":
		*m = ScanMode_ScanModeFull
	case "accelerated":
		*m = ScanMode_ScanModeAccelerated
	default:
		return errors.Errorf("unknown scan mode specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular scan mode is a valid,
// non-default value.
func (m ScanMode) Supported() bool {
	switch m {
	case ScanMode_ScanModeFull:
		return true
	case ScanMode_ScanModeAccelerated:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a scan mode.
func (m ScanMode) Description() string {
	switch m {
	case ScanMode_ScanModeDefault:
		return "Default"
	case ScanMode_ScanModeFull:
		return "Full"
	case ScanMode_ScanModeAccelerated:
		return "Accelerated"
	default:
		return "Unknown"
	}
}
