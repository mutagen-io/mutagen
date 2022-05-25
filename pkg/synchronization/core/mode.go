package core

import (
	"fmt"
)

// IsDefault indicates whether or not the synchronization mode is
// SynchronizationMode_SynchronizationModeDefault.
func (m SynchronizationMode) IsDefault() bool {
	return m == SynchronizationMode_SynchronizationModeDefault
}

// MarshalText implements encoding.TextMarshaler.MarshalText.
func (m SynchronizationMode) MarshalText() ([]byte, error) {
	var result string
	switch m {
	case SynchronizationMode_SynchronizationModeDefault:
	case SynchronizationMode_SynchronizationModeTwoWaySafe:
		result = "two-way-safe"
	case SynchronizationMode_SynchronizationModeTwoWayResolved:
		result = "two-way-resolved"
	case SynchronizationMode_SynchronizationModeOneWaySafe:
		result = "one-way-safe"
	case SynchronizationMode_SynchronizationModeOneWayReplica:
		result = "one-way-replica"
	default:
		result = "unknown"
	}
	return []byte(result), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText.
func (m *SynchronizationMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a synchronization mode.
	switch text {
	case "two-way-safe":
		*m = SynchronizationMode_SynchronizationModeTwoWaySafe
	case "two-way-resolved":
		*m = SynchronizationMode_SynchronizationModeTwoWayResolved
	case "one-way-safe":
		*m = SynchronizationMode_SynchronizationModeOneWaySafe
	case "one-way-replica":
		*m = SynchronizationMode_SynchronizationModeOneWayReplica
	default:
		return fmt.Errorf("unknown synchronization mode specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular synchronization mode is a
// valid, non-default value.
func (m SynchronizationMode) Supported() bool {
	switch m {
	case SynchronizationMode_SynchronizationModeTwoWaySafe:
		return true
	case SynchronizationMode_SynchronizationModeTwoWayResolved:
		return true
	case SynchronizationMode_SynchronizationModeOneWaySafe:
		return true
	case SynchronizationMode_SynchronizationModeOneWayReplica:
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
	case SynchronizationMode_SynchronizationModeTwoWaySafe:
		return "Two Way Safe"
	case SynchronizationMode_SynchronizationModeTwoWayResolved:
		return "Two Way Resolved"
	case SynchronizationMode_SynchronizationModeOneWaySafe:
		return "One Way Safe"
	case SynchronizationMode_SynchronizationModeOneWayReplica:
		return "One Way Replica"
	default:
		return "Unknown"
	}
}
