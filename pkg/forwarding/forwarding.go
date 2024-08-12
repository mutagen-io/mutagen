package forwarding

import (
	"context"
	"io"
	"net"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
	"github.com/mutagen-io/mutagen/pkg/stream"
)

// ForwardAndClose performs bidirectional forwarding between the specified
// connections. It waits for both directions to see EOF, for one direction to
// see an error, or for context cancellation. Once one of these events occurs,
// the connections are closed (terminating forwarding) and the function returns.
// Both connections must implement CloseWriter or this function will panic. If
// the caller passes non-nil values for firstAuditor and/or secondAuditor, then
// auditing will be performed on the write end of the respective connection.
func ForwardAndClose(ctx context.Context, first, second net.Conn, firstAuditor, secondAuditor stream.Auditor, logger *logging.Logger) {
	// Defer closure of the connections.
	defer func() {
		must.Close(first, logger)
		must.Close(second, logger)
	}()

	// Extract write closure interfaces.
	firstCloseWriter, ok := first.(stream.CloseWriter)
	if !ok {
		panic("first connection does not implement write closure")
	}
	secondCloseWriter, ok := second.(stream.CloseWriter)
	if !ok {
		panic("second connection does not implement write closure")
	}

	// Forward traffic between the connections (with optional auditing) in
	// separate Goroutines and track their termination. We track their
	// termination via the error result, though this may be nil in the event
	// that the source indicates EOF. If we do see an EOF from a source, then
	// perform write closure on the corresponding destination in order to
	// forward the EOF.
	copyErrors := make(chan error, 2)
	go func() {
		_, err := io.Copy(stream.NewAuditWriter(first, firstAuditor), second)
		if err == nil {
			must.CloseWrite(firstCloseWriter, logger)
		}
		copyErrors <- err
	}()
	go func() {
		_, err := io.Copy(stream.NewAuditWriter(second, secondAuditor), first)
		if err == nil {
			must.CloseWrite(secondCloseWriter, logger)
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
