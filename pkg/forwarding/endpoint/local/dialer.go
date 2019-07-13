package local

import (
	"context"
	"net"

	"github.com/havoc-io/mutagen/pkg/forwarding"
)

// dialerEndpoint implements forwarding.Endpoint for dialer endpoints.
type dialerEndpoint struct {
	// dialingContext is the context that governs dialing operations.
	dialingContext context.Context
	// dialingCancel is the context cancellation function that cancels the
	// dialing context.
	dialingCancel context.CancelFunc
	// dialer is the underlying dialer.
	dialer *net.Dialer
	// protocol is the protocol to use for dialing.
	protocol string
	// address is the address to use for dialing.
	address string
}

// NewDialerEndpoint creates a new forwarding.Endpoint that behaves as a
// dialer.
func NewDialerEndpoint(protocol, address string) (forwarding.Endpoint, error) {
	// Create a cancellable context that we can use to regulate connections.
	dialingContext, dialingCancel := context.WithCancel(context.Background())

	// Create the endpoint.
	return &dialerEndpoint{
		dialingContext: dialingContext,
		dialingCancel:  dialingCancel,
		dialer:         &net.Dialer{},
		protocol:       protocol,
		address:        address,
	}, nil
}

// Shutdown implements forwarding.Endpoint.Open.
func (e *dialerEndpoint) Open() (net.Conn, error) {
	return e.dialer.DialContext(e.dialingContext, e.protocol, e.address)
}

// Shutdown implements forwarding.Endpoint.Shutdown.
func (e *dialerEndpoint) Shutdown() error {
	// Cancel the dialing context to unblock any dialing operations.
	e.dialingCancel()

	// Success.
	return nil
}
