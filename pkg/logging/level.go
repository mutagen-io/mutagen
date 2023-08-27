package logging

import (
	"strings"
)

// Level represents a log level. Its value hierarchy is designed to be ordered
// and comparable by value.
type Level uint

const (
	// LevelDisabled indicates that logging is completely disabled.
	LevelDisabled Level = iota
	// LevelError indicates that only fatal errors are logged.
	LevelError
	// LevelWarn indicates that both fatal and non-fatal errors are logged.
	LevelWarn
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

// levelNames are the human-readable representations of log levels.
var levelNames = [6]string{
	"disabled",
	"error",
	"warn",
	"info",
	"debug",
	"trace",
}

// String provides a human-readable representation of a log level.
func (l Level) String() string {
	if l <= LevelTrace {
		return levelNames[l]
	}
	return "unknown"
}

// NameToLevel converts a string-based representation of a log level to the
// appropriate Level value. It returns a boolean indicating whether or not the
// conversion was valid. If the name is invalid, LevelDisabled is returned.
func NameToLevel(name string) (Level, bool) {
	switch name {
	case "disabled":
		return LevelDisabled, true
	case "error":
		return LevelError, true
	case "warn":
		return LevelWarn, true
	case "info":
		return LevelInfo, true
	case "debug":
		return LevelDebug, true
	case "trace":
		return LevelTrace, true
	default:
		return LevelDisabled, false
	}
}

// abbreviations is the range of abbreviations to use for log levels.
const abbreviations = "_EWIDT"

// abbreviation returns a one-byte prefix to use for the level in log lines.
func (l Level) abbreviation() byte {
	if l <= LevelTrace {
		return abbreviations[l]
	}
	return '?'
}

// abbreviationToLevel converts a one-byte prefix representation of a log level
// to the appropriate Level value. It returns a boolean indicating whether or
// not the conversion was valid. If the abbreviation is invalid, LevelDisabled
// is returned.
func abbreviationToLevel(abbreviation byte) (Level, bool) {
	if index := strings.IndexByte(abbreviations, abbreviation); index != -1 {
		return Level(index), true
	}
	return LevelDisabled, false
}
