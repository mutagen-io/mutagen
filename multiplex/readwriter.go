package multiplex

import (
	"io"
)

type readWriter struct {
	io.Reader
	io.Writer
}

// ReadWriter performs multiplexing of a duplex stream. It is a simple wrapper
// around the Reader and Writer methods, and thus the underlying stream must
// adhere to the requirements of the arguments to those methods (which you
// should review before using this method).
func ReadWriter(stream io.ReadWriter, channels uint8) ([]io.ReadWriter, io.Closer) {
	// Perform read multiplexing.
	readers, closer := Reader(stream, channels)

	// Perform write multiplexing.
	writers := Writer(stream, channels)

	// Join streams.
	streams := make([]io.ReadWriter, channels)
	for c := uint8(0); c < channels; c++ {
		streams[c] = &readWriter{readers[c], writers[c]}
	}

	// Done.
	return streams, closer
}
