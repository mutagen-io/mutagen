package logging

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/mutagen-io/mutagen/pkg/stream"
)

// Logger is the main logger type. A nil Logger is valid and all of its methods
// are no-ops. It is safe for concurrent usage and will serialize access to the
// underlying writer.
type Logger struct {
	// level is the log level.
	level Level
	// scope is the logger's scope.
	scope string
	// writer is the underlying writer.
	writer io.Writer
}

// NewLogger creates a new logger at the specified log level targeting the
// specified writer. The writer must be non-nil. The logger and any derived
// subloggers will coordinate access to the writer.
func NewLogger(level Level, writer io.Writer) *Logger {
	return &Logger{
		level:  level,
		writer: stream.NewConcurrentWriter(writer),
	}
}

// Level returns the logger's log level. It can be used to restrict certain
// computations to cases where their results will actually be used, for example
// statistics that only need to be calculated when debugging.
func (l *Logger) Level() Level {
	// If the logger is nil, then logging is disabled.
	if l == nil {
		return LevelDisabled
	}

	// Return the log level.
	return l.level
}

// nameMatcher is used to validate names passed to Sublogger.
var nameMatcher = regexp.MustCompile("^[[:word:]]+$")

// Sublogger creates a new sublogger with the specified name. Names must be
// non-empty and may only contain the characters a-z, A-Z, 0-9, and underscores.
// Attempts to use an invalid name will result in a nil logger and a warning
// being issued on the current logger.
func (l *Logger) Sublogger(name string) *Logger {
	// If the logger is nil, then the sublogger will be as well.
	if l == nil {
		return nil
	}

	// Validate the sublogger name.
	if !nameMatcher.MatchString(name) {
		l.Warn("attempt to create sublogger with invalid name")
		return nil
	}

	// Compute the new logger's scope.
	scope := name
	if l.scope != "" {
		scope = l.scope + "." + scope
	}

	// Create the new logger.
	return &Logger{
		level:  l.level,
		scope:  scope,
		writer: l.writer,
	}
}

// timestampFormat is the format in which timestamps should be rendered.
const timestampFormat = "2006-01-02 15:04:05.000000"

// write writes a log message to the underlying writer.
func (l *Logger) write(timestamp time.Time, level Level, message string) {
	// If a carriage return is found, then truncate the message at that point.
	if index := strings.IndexByte(message, '\r'); index >= 0 {
		message = message[:index] + "...\n"
	}

	// Ensure that the only newline character in the message appears at the end
	// of the string. If one appears earlier, then truncate the message at that
	// point. If none appears, then something has gone wrong with formatting.
	if index := strings.IndexByte(message, '\n'); index < 0 {
		panic("no newline character found in formatted message")
	} else if index != len(message)-1 {
		message = message[:index] + "...\n"
	}

	// Compute the log line.
	var line string
	if l.scope != "" {
		line = fmt.Sprintf("%s [%c] [%s] %s",
			timestamp.Format(timestampFormat), level.abbreviation(), l.scope, message,
		)
	} else {
		line = fmt.Sprintf("%s [%c] %s",
			timestamp.Format(timestampFormat), level.abbreviation(), message,
		)
	}

	// Write the line. We can't do much with the error here, so we don't try.
	// Practically speaking, most io.Writer implementations perform retries if a
	// short write occurs, so retrying here (on top of that logic) probably
	// wouldn't help much. Even if we wanted to, we'd be better off wrapping the
	// writer in a hypothetical RetryingWriter in order to better encapsulate
	// that logic and to avoid having to add a lock outside the writer. In any
	// case, Go's standard log package also discards analogous errors, so we'll
	// do the same for the time being.
	l.writer.Write([]byte(line))
}

// log provides logging with formatting semantics equivalent to fmt.Sprintln.
func (l *Logger) log(level Level, v ...any) {
	if l != nil && l.level >= level {
		l.write(time.Now(), level, fmt.Sprintln(v...))
	}
}

// logf provides logging with formatting semantics equivalent to fmt.Sprintf. It
// automatically appends a trailing newline to the format string.
func (l *Logger) logf(level Level, format string, v ...any) {
	if l != nil && l.level >= level {
		l.write(time.Now(), level, fmt.Sprintf(format+"\n", v...))
	}
}

// Error logs errors with formatting semantics equivalent to fmt.Sprintln.
func (l *Logger) Error(v ...any) {
	l.log(LevelError, v...)
}

// Errorf logs errors with formatting semantics equivalent to fmt.Sprintf. A
// trailing newline is automatically appended and should not be included in the
// format string.
func (l *Logger) Errorf(format string, v ...any) {
	l.logf(LevelError, format, v...)
}

// Warn logs warnings with formatting semantics equivalent to fmt.Sprintln.
func (l *Logger) Warn(v ...any) {
	l.log(LevelWarn, v...)
}

// Warnf logs warnings with formatting semantics equivalent to fmt.Sprintf. A
// trailing newline is automatically appended and should not be included in the
// format string.
func (l *Logger) Warnf(format string, v ...any) {
	l.logf(LevelWarn, format, v...)
}

// Info logs information with formatting semantics equivalent to fmt.Sprintln.
func (l *Logger) Info(v ...any) {
	l.log(LevelInfo, v...)
}

// Infof logs information with formatting semantics equivalent to fmt.Sprintf. A
// trailing newline is automatically appended and should not be included in the
// format string.
func (l *Logger) Infof(format string, v ...any) {
	l.logf(LevelInfo, format, v...)
}

// Debug logs debug information with formatting semantics equivalent to
// fmt.Sprintln.
func (l *Logger) Debug(v ...any) {
	l.log(LevelDebug, v...)
}

// Debugf logs debug information with formatting semantics equivalent to
// fmt.Sprintf. A trailing newline is automatically appended and should not be
// included in the format string.
func (l *Logger) Debugf(format string, v ...any) {
	l.logf(LevelDebug, format, v...)
}

// Trace logs tracing information with formatting semantics equivalent to
// fmt.Sprintln.
func (l *Logger) Trace(v ...any) {
	l.log(LevelTrace, v...)
}

// Tracef logs tracing information with formatting semantics equivalent to
// fmt.Sprintf. A trailing newline is automatically appended and should not be
// included in the format string.
func (l *Logger) Tracef(format string, v ...any) {
	l.logf(LevelTrace, format, v...)
}

// linePrefixMatcher matches the timestamp and level prefix of logging lines.
var linePrefixMatcher = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{6} \[([` + abbreviations + `])\] `)

// Writer returns an io.Writer that logs incoming lines. If an incoming line is
// determined to be an output line from another logger, then it will be parsed
// and gated against this logger's level, its scope will be merged with that of
// this logger, and the combined line will be written. Otherwise, if an incoming
// line is not determined to be from another logger, than it will be written as
// a message with the specificed level.
//
// Note that unlike the Logger itself, the writer returned from this method is
// not safe for concurrent use by multiple Goroutines. An external locking
// mechanism should be added if concurrent use is necessary.
func (l *Logger) Writer(level Level) io.Writer {
	// If the current logger is nil, then we can just discard all output.
	if l == nil {
		return io.Discard
	}

	// Create the writer.
	return &stream.LineProcessor{
		Callback: func(line string) {
			// Check if the line is output from a logger. If it's not, then we
			// just log it as if it were any other message.
			matches := linePrefixMatcher.FindStringSubmatch(line)
			if len(matches) != 2 {
				l.log(level, line)
				return
			}

			// Decode the log level for the line. If the log level that it
			// specifies is invalid, then just print an indicator that an
			// invalid line was received. Otherwise, if the line level is beyond
			// the threshold of this logger, then just ignore it.
			if len(matches[1]) != 1 {
				panic("line prefix matcher returned invalid match")
			} else if level, ok := abbreviationToLevel(matches[1][0]); !ok {
				l.Warn("<invalid incoming log line level>")
				return
			} else if l.level < level {
				return
			}

			// If we have a non-empty scope, then inject it into the line. If
			// not, then just add (back) the newline character.
			if l.scope != "" {
				line = fmt.Sprintf("%s[%s] %s\n", matches[0], l.scope, line[len(matches[0]):])
			} else {
				line = line + "\n"
			}

			// Write the line to the underlying writer.
			l.writer.Write([]byte(line))
		},
	}
}
