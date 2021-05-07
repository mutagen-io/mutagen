package logging

import (
	"fmt"
	"io"
	"log"

	"github.com/mutagen-io/mutagen/pkg/stream"
)

// Logger performs level-based logging. It is a wrapper around the standard log
// package's Logger type, but with more flexible levels and without the ability
// to terminate the program. A nil Logger is valid for use and simply performs
// no logging and returns only nil subloggers. Loggers are safe for concurrent
// usage by multiple Goroutines.
type Logger struct {
	// logger is the shared underlying logger.
	logger *log.Logger
	// prefix is any prefix specified for the logger.
	prefix string
}

// NewLogger constructs a new root-level logger (i.e. a logger with no message
// prefix) using the specified output stream. Loggers for subsystems should be
// created using the Sublogger method. Write access to the output stream is
// automatically serialized.
func NewLogger(output io.Writer) *Logger {
	return &Logger{
		logger: log.New(output, "", log.Ldate|log.Lmicroseconds|log.LUTC),
	}
}

// Sublogger creates a new sublogger with the specified name.
func (l *Logger) Sublogger(name string) *Logger {
	// If the logger is nil, then the sublogger will be as well.
	if l == nil {
		return nil
	}

	// Compute the new prefix.
	prefix := name
	if l.prefix != "" {
		prefix = l.prefix + "." + name
	}

	// Create the new logger.
	return &Logger{
		logger: l.logger,
		prefix: prefix,
	}
}

// output is the shared internal logging method.
func (l *Logger) output(level, line string) {
	// Compute the formatted line.
	if l.prefix != "" {
		line = fmt.Sprintf("[%s|%s] %s", l.prefix, level, line)
	} else {
		line = fmt.Sprintf("[%s] %s", level, line)
	}

	// Log.
	l.logger.Output(4, line)
}

// println provides logging with formatting semantics equivalent to fmt.Println.
func (l *Logger) println(level Level, v ...interface{}) {
	if l != nil && currentLevel >= level {
		l.output(level.String(), fmt.Sprintln(v...))
	}
}

// printf provides logging with formatting semantics equivalent to fmt.Printf.
func (l *Logger) printf(level Level, format string, v ...interface{}) {
	if l != nil && currentLevel >= level {
		l.output(level.String(), fmt.Sprintf(format, v...))
	}
}

// Error logs errors with formatting semantics equivalent to fmt.Println.
func (l *Logger) Error(v ...interface{}) {
	l.println(LevelError, v...)
}

// Errorf logs errors with formatting semantics equivalent to fmt.Printf.
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.printf(LevelError, format, v...)
}

// Warning logs warnings with formatting semantics equivalent to fmt.Println.
func (l *Logger) Warning(v ...interface{}) {
	l.println(LevelWarning, v...)
}

// Warningf logs warnings with formatting semantics equivalent to fmt.Printf.
func (l *Logger) Warningf(format string, v ...interface{}) {
	l.printf(LevelWarning, format, v...)
}

// Info logs information with formatting semantics equivalent to fmt.Println.
func (l *Logger) Info(v ...interface{}) {
	l.println(LevelInfo, v...)
}

// Infof logs information with formatting semantics equivalent to fmt.Printf.
func (l *Logger) Infof(format string, v ...interface{}) {
	l.printf(LevelInfo, format, v...)
}

// Debug logs debug information with formatting semantics equivalent to
// fmt.Println.
func (l *Logger) Debug(v ...interface{}) {
	l.println(LevelDebug, v...)
}

// Debugf logs debug information with formatting semantics equivalent to
// fmt.Printf.
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.printf(LevelDebug, format, v...)
}

// Trace logs tracing information with formatting semantics equivalent to
// fmt.Println.
func (l *Logger) Trace(v ...interface{}) {
	l.println(LevelTrace, v...)
}

// Tracef logs tracing information with formatting semantics equivalent to
// fmt.Printf.
func (l *Logger) Tracef(format string, v ...interface{}) {
	l.printf(LevelTrace, format, v...)
}

// Writer returns an io.Writer that logs output lines using the specified level.
func (l *Logger) Writer(level Level) io.Writer {
	// If the logger is nil or the current logging level is set lower than the
	// requested level, then we can just discard input since it won't be logged
	// anyway. This saves us the overhead of scanning lines.
	if l == nil || currentLevel < level {
		return io.Discard
	}

	// Create the writer.
	return &stream.LineProcessor{
		Callback: func(s string) {
			l.println(level, s)
		},
	}
}
