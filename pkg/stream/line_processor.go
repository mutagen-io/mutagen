package stream

import (
	"bytes"
	"errors"
)

const (
	// defaultLineProcessorMaximumBufferSize is the default maximum buffer size
	// for LineProcessor.
	defaultLineProcessorMaximumBufferSize = 64 * 1024
)

// ErrMaximumBufferSizeExceeded is returned when a write would exceed the
// maximum internal buffer size for a writer.
var ErrMaximumBufferSizeExceeded = errors.New("maximum buffer size exceed")

// trimCarriageReturn trims any single trailing carriage return from the end of
// a byte slice.
func trimCarriageReturn(buffer []byte) []byte {
	if len(buffer) > 0 && buffer[len(buffer)-1] == '\r' {
		return buffer[:len(buffer)-1]
	}
	return buffer
}

// LineProcessor is an io.Writer that splits its input stream into lines and
// writes those lines to a callback function. Line splits are performed on any
// instance of '\n' or '\r\n', with the split character(s) removed from the
// callback value.
type LineProcessor struct {
	// Callback is the line processing callback.
	Callback func(string)
	// MaximumBufferSize is the maximum allowed internal buffer size. If writes
	// to the writer exceed this size without incorporating a newline, then an
	// error will be raised. A value of 0 causes the writer to use a reasonable
	// default. A negative value indicates no limit.
	MaximumBufferSize int
	// buffer is any incomplete line fragment left over from a previous write.
	buffer []byte
}

// Write implements io.Writer.Write.
func (p *LineProcessor) Write(data []byte) (int, error) {
	// Ensure that storing the data won't exceed buffer size limits.
	if p.MaximumBufferSize == 0 && len(p.buffer)+len(data) > defaultLineProcessorMaximumBufferSize {
		return 0, ErrMaximumBufferSizeExceeded
	} else if p.MaximumBufferSize > 0 && len(p.buffer)+len(data) > p.MaximumBufferSize {
		return 0, ErrMaximumBufferSizeExceeded
	}

	// Append the data to our internal buffer.
	p.buffer = append(p.buffer, data...)

	// Process all lines in the buffer and track the number of processed bytes.
	var processed int
	remaining := p.buffer
	for {
		// Find the index of the next newline character.
		index := bytes.IndexByte(remaining, '\n')
		if index == -1 {
			break
		}

		// Process the line.
		p.Callback(string(trimCarriageReturn(remaining[:index])))

		// Update the number of bytes that we've processed.
		processed += index + 1

		// Update the remaining slice.
		remaining = remaining[index+1:]
	}

	// If we managed to process bytes, then truncate our internal buffer.
	if processed > 0 {
		// Compute the number of leftover bytes.
		leftover := len(p.buffer) - processed

		// If there are leftover bytes, then shift them to the front of the
		// buffer.
		if leftover > 0 {
			copy(p.buffer[:leftover], p.buffer[processed:])
		}

		// Truncate the buffer.
		p.buffer = p.buffer[:leftover]
	}

	// Done.
	return len(data), nil
}
