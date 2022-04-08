package logging

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

// String provides a human-readable representation of a log level.
func (l Level) String() string {
	switch l {
	case LevelDisabled:
		return "disabled"
	case LevelError:
		return "error"
	case LevelWarn:
		return "warn"
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
