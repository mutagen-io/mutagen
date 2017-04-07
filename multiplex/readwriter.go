package multiplex

import (
	"io"
)

// readWriter is an internal type to join an io.Reader and io.Writer into a
// single io.ReadWriter.
type readWriter struct {
	io.Reader
	io.Writer
}

// ReadWriter performs multiplexing of a duplex stream. The returned closer is
// that returned by multiplexing the io.Reader portion of the interface with the
// Reader function, so it follows the same semantics.
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
