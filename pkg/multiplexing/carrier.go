package multiplexing

import (
	"bufio"
	"io"
)

// Carrier is the interface that the streams used for multiplexing must
// implement. It imposes the additional constraint that the Close method must
// unblock any pending read, discard, or write operations. This interface can be
// implemented by custom code, but it can also be implemented efficiently using
// the NewCarrierFromStream function.
type Carrier interface {
	io.Reader
	io.ByteReader
	// Discard attempts to discard the next n bytes from the stream, returning
	// the number of bytes discarded and any error that occurred. The returned
	// error must be non-nil if and only if discarded != n.
	Discard(n int) (discarded int, err error)
	io.Writer
	io.Closer
}

// bufioCarrier is a Carrier implementation that can be used to adapt an
// underlying io.ReadWriteCloser to a Carrier.
type bufioCarrier struct {
	*bufio.Reader
	io.Writer
	io.Closer
}

// NewCarrierFromStream constructs a new Carrier by wrapping an underlying
// io.ReadWriteCloser. The underlying stream must have the property that its
// Close method unblocks any pending Read or Write calls.
func NewCarrierFromStream(stream io.ReadWriteCloser) Carrier {
	return &bufioCarrier{
		bufio.NewReader(stream),
		stream,
		stream,
	}
}
