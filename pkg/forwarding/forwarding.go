package forwarding

import (
	"context"
	"io"
	"net"
)

// CloseWriter is the interface for connections implementing write closure.
type CloseWriter interface {
	// CloseWrite should close the stream for further writes. It need not
	// unblock pending writes, but it should prevent future writes and send an
	// EOF indicator to the read end of the other end of the connection.
	CloseWrite() error
}

// ForwardAndClose performs bidirectional forwarding between the specified
// connections. It waits for both directions to see EOF, for one direction to
// see an error, or for context cancellation. Once one of these events occurs,
// the connections are closed (terminating forwarding) and the function returns.
// Both connections must implement CloseWriter or this function will panic.
func ForwardAndClose(ctx context.Context, first, second net.Conn) {
	// Defer closure of the connections.
	defer func() {
		first.Close()
		second.Close()
	}()

	// Extract write closure interfaces.
	firstCloseWriter, ok := first.(CloseWriter)
	if !ok {
		panic("first connection does not implement write closure")
	}
	secondCloseWriter, ok := second.(CloseWriter)
	if !ok {
		panic("second connection does not implement write closure")
	}

	// Forward traffic between the connections in separate Goroutines and track
	// their termination. We track their termination via the error result,
	// though this may be nil in the event that the source indicates EOF. If we
	// do see an EOF from a source, then perform write closure on the
	// corresponding destination in order to forward the EOF.
	copyErrors := make(chan error, 2)
	go func() {
		_, err := io.Copy(first, second)
		if err == nil {
			firstCloseWriter.CloseWrite()
		}
		copyErrors <- err
	}()
	go func() {
		_, err := io.Copy(second, first)
		if err == nil {
			secondCloseWriter.CloseWrite()
		}
		copyErrors <- err
	}()

	// Wait for both forwarding routines to finish while also monitoring for
	// termination. We only abort this wait if we see a non-nil copy error from
	// one of the forwarding routines (or forwarding is terminated). We allow
	// nil errors because they simply indicate EOF and can be sent by some
	// connection types by performing a half-close of a stream.
	for i := 0; i < 2; i++ {
		select {
		case err := <-copyErrors:
			if err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
