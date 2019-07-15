package logging

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/fatih/color"

	"github.com/mutagen-io/mutagen/pkg/mutagen"
)

// writer is an io.Writer that splits its input stream into lines and writes
// those lines to an underlying logger.
type writer struct {
	// callback is the logging callback.
	callback func(string)
	// buffer is any incomplete line fragment left over from a previous write.
	buffer []byte
}

// trimCarriageReturn trims any single trailing carriage return from the end of
// a byte slice.
func trimCarriageReturn(buffer []byte) []byte {
	if len(buffer) > 0 && buffer[len(buffer)-1] == '\r' {
		return buffer[:len(buffer)-1]
	}
	return buffer
}

// Write implements io.Writer.Write.
func (w *writer) Write(buffer []byte) (int, error) {
	// Append the data to our internal buffer.
	w.buffer = append(w.buffer, buffer...)

	// Process all lines in the buffer, tracking the number of bytes that we
	// process.
	var processed int
	remaining := w.buffer
	for {
		// Find the index of the next newline character.
		index := bytes.IndexByte(remaining, '\n')
		if index == -1 {
			break
		}

		// Process the line.
		w.callback(string(trimCarriageReturn(remaining[:index])))

		// Update the number of bytes that we've processed.
		processed += index + 1

		// Update the remaining slice.
		remaining = remaining[index+1:]
	}

	// If we managed to process bytes, then truncate our internal buffer.
	if processed > 0 {
		// Compute the number of leftover bytes.
		leftover := len(w.buffer) - processed

		// If there are leftover bytes, then shift them to the front of the
		// buffer.
		if leftover > 0 {
			copy(w.buffer[:leftover], w.buffer[processed:])
		}

		// Truncate the buffer.
		w.buffer = w.buffer[:leftover]
	}

	// Done.
	return len(buffer), nil
}

// Logger is the main logger type. It has the novel property that it still
// functions if nil, but it doesn't log anything. It is designed to use the
// standard logger provided by the log package, so it respects any flags set for
// that logger. It is safe for concurrent usage.
type Logger struct {
	// prefix is any prefix specified for the logger.
	prefix string
}

// RootLogger is the root logger from which all other loggers derive.
var RootLogger = &Logger{}

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
		prefix: prefix,
	}
}

// output is the internal logging method.
func (l *Logger) output(calldepth int, line string) {
	// Add a prefix if necessary.
	if l.prefix != "" {
		line = fmt.Sprintf("[%s] %s", l.prefix, line)
	}

	// Log.
	log.Output(calldepth, line)
}

// Print logs information with semantics equivalent to fmt.Print.
func (l *Logger) Print(v ...interface{}) {
	if l != nil {
		l.output(3, fmt.Sprint(v...))
	}
}

// Printf logs information with semantics equivalent to fmt.Printf.
func (l *Logger) Printf(format string, v ...interface{}) {
	if l != nil {
		l.output(3, fmt.Sprintf(format, v...))
	}
}

// Println logs information with semantics equivalent to fmt.Println.
func (l *Logger) Println(v ...interface{}) {
	if l != nil {
		l.output(3, fmt.Sprintln(v...))
	}
}

// Writer returns an io.Writer that writes lines using Println.
func (l *Logger) Writer() io.Writer {
	// If the logger is nil, then we can just discard input since it won't be
	// logged anyway. This saves us the overhead of scanning lines.
	if l == nil {
		return ioutil.Discard
	}

	// Create the writer.
	return &writer{
		callback: func(s string) {
			l.Println(s)
		},
	}
}

// Debug logs information with semantics equivalent to fmt.Print, but only if
// debugging is enabled (otherwise it's a no-op).
func (l *Logger) Debug(v ...interface{}) {
	if l != nil && mutagen.DebugEnabled {
		l.output(3, fmt.Sprint(v...))
	}
}

// Debugf logs information with semantics equivalent to fmt.Printf, but only if
// debugging is enabled (otherwise it's a no-op).
func (l *Logger) Debugf(format string, v ...interface{}) {
	if l != nil && mutagen.DebugEnabled {
		l.output(3, fmt.Sprintf(format, v...))
	}
}

// Debugln logs information with semantics equivalent to fmt.Println, but only
// if debugging is enabled (otherwise it's a no-op).
func (l *Logger) Debugln(v ...interface{}) {
	if l != nil && mutagen.DebugEnabled {
		l.output(3, fmt.Sprintln(v...))
	}
}

// DebugWriter returns an io.Writer that writes lines using Debugln.
func (l *Logger) DebugWriter() io.Writer {
	// If the logger is nil, then we can just discard input since it won't be
	// logged anyway. This saves us the overhead of scanning lines.
	if l == nil {
		return ioutil.Discard
	}

	// Create the writer.
	return &writer{
		callback: func(s string) {
			l.Debugln(s)
		},
	}
}

// Warn logs error information with a warning prefix and yellow color.
func (l *Logger) Warn(err error) {
	if l != nil {
		l.output(3, color.YellowString("Warning: %v", err))
	}
}

// Error logs error information with an error prefix and red color.
func (l *Logger) Error(err error) {
	if l != nil {
		l.output(3, color.RedString("Error: %v", err))
	}
}
