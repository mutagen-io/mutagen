package synchronization

import (
	"fmt"
)

// IsDefault indicates whether or not the scan mode is ScanMode_ScanModeDefault.
func (m ScanMode) IsDefault() bool {
	return m == ScanMode_ScanModeDefault
}

// MarshalText implements encoding.TextMarshaler.MarshalText.
func (m ScanMode) MarshalText() ([]byte, error) {
	var result string
	switch m {
	case ScanMode_ScanModeDefault:
	case ScanMode_ScanModeFull:
		result = "full"
	case ScanMode_ScanModeAccelerated:
		result = "accelerated"
	default:
		result = "unknown"
	}
	return []byte(result), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText.
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
		return fmt.Errorf("unknown scan mode specification: %s", text)
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
