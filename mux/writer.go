package mux

import (
	"bufio"
	"io"
	"sync"

	"github.com/pkg/errors"
)

// writeMultiplexer implements the multiplexing protocol over an io.Writer.
type writeMultiplexer struct {
	// lock restricts access to the multiplexer's state.
	lock sync.Mutex
	// writer is a buffered version of the underlying stream. Technically we
	// don't need the stream to be buffered, but we mandate that it be buffered
	// to avoid unnecessary overhead on short writes during header encoding.
	writer *bufio.Writer
	// error is the last error that occurred during a write.
	error error
}

// write performs a header-prefixed write for a channel.
func (m *writeMultiplexer) write(channel uint8, buffer []byte) (int, error) {
	// Check if the buffer is a size we can handle. If it's not, this isn't an
	// error that will prevent subsequent writes, so we don't record it.
	// TODO: Ideally we'd chunk the buffer and write each block individually,
	// but I don't think we're likely to exceed 4 GB writes at the moment, so
	// I'll punt on that.
	if len(buffer) > maxBlockLength {
		return 0, errors.New("write size too big for multiplexing")
	}

	// Lock the writer and defer its release.
	m.lock.Lock()
	defer m.lock.Unlock()

	// Check if the multiplexer is errored.
	if m.error != nil {
		return 0, errors.Wrap(m.error, "previous write error encountered")
	}

	// Create and write a header.
	header := header{channel, uint32(len(buffer))}
	if err := header.write(m.writer); err != nil {
		m.error = errors.Wrap(err, "unable to write header")
		return 0, m.error
	}

	// Write the data.
	n, err := m.writer.Write(buffer)
	if err != nil {
		m.error = err
		return n, err
	}

	// Flush any buffered data. It's a bit weird to return n here in the case of
	// failure, but it's not disallowed by the io.Writer interface (the only
	// requirement is that the error be non-nil on short writes).
	if err = m.writer.Flush(); err != nil {
		m.error = errors.Wrap(err, "unable to flush buffered data")
		return n, m.error
	}

	// Success.
	return n, nil
}

// writeStream implements io.Writer for a single multiplexing channel.
type writeStream struct {
	// multiplexer is the underlying write multiplexer. It is shared by all
	// channels being multiplexed over a particular stream.
	multiplexer *writeMultiplexer
	// channel is the index for this multiplexing channel.
	channel uint8
}

// Write sends data on the multiplexing channel.
func (s *writeStream) Write(buffer []byte) (int, error) {
	return s.multiplexer.write(s.channel, buffer)
}

// Writer multiplexes an io.Writer into independent byte streams that can be
// demultiplexed by the Reader method.
func Writer(writer io.Writer, channels uint8) []io.Writer {
	// Create the shared multiplexer.
	multiplexer := &writeMultiplexer{
		writer: bufio.NewWriter(writer),
	}

	// Create the individual writer streams.
	streams := make([]io.Writer, channels)
	for c := uint8(0); c < channels; c++ {
		streams[c] = &writeStream{multiplexer, c}
	}

	// Done.
	return streams
}
