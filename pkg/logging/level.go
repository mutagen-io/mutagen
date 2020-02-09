package logging

import (
	"os"
)

// Level represents a logging level.
type Level uint8

const (
	// LevelDisabled indicates that logging is completely disabled.
	LevelDisabled Level = iota
	// LevelError indicates that only fatal errors are logged.
	LevelError
	// LevelWarning indicates that both fatal and non-fatal errors are logged.
	LevelWarning
	// LevelInfo indicates that basic execution information is logged (in
	// addition to all errors).
	LevelInfo
	// LevelDebug indicates that advanced execution information is logged (in
	// addition to basic information and all errors).
	LevelDebug
	// LevelTrace indicates that low-level execution information is logged (in
	// addition to all other execution information and all errors).
	LevelTrace
)

// String provides a human-readable representation of a logging level.
func (l Level) String() string {
	switch l {
	case LevelDisabled:
		return "disabled"
	case LevelError:
		return "error"
	case LevelWarning:
		return "warning"
	case LevelInfo:
		return "info"
	case LevelDebug:
		return "debug"
	case LevelTrace:
		return "trace"
	default:
		return "unknown"
	}
}

// currentLevel is the current logging level.
var currentLevel Level

func init() {
	// Set the log level based on the MUTAGEN_LOG_LEVEL environment variable. If
	// unset (or set to an unknown value), then default to LevelInfo.
	switch os.Getenv("MUTAGEN_LOG_LEVEL") {
	case "disabled":
		currentLevel = LevelDisabled
	case "error":
		currentLevel = LevelError
	case "warning":
		currentLevel = LevelWarning
	case "info":
		currentLevel = LevelInfo
	case "debug":
		currentLevel = LevelDebug
	case "trace":
		currentLevel = LevelTrace
	default:
		currentLevel = LevelInfo
	}
}
