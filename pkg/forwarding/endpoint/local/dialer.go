package local

import (
	"context"
	"net"

	"github.com/mutagen-io/mutagen/pkg/forwarding"
)

// dialerEndpoint implements forwarding.Endpoint for dialer endpoints.
type dialerEndpoint struct {
	// dialingContext limits the duration of dialing operations.
	dialingContext context.Context
	// dialingCancel cancels the dialing context.
	dialingCancel context.CancelFunc
	// dialer is the dialer used for TCP and Unix domain socket dialing.
	dialer *net.Dialer
	// protocol is the protocol to use for dialing.
	protocol string
	// address is the address to use for dialing.
	address string
}

// NewDialerEndpoint creates a new forwarding.Endpoint that acts as a dialer.
func NewDialerEndpoint(
	version forwarding.Version,
	configuration *forwarding.Configuration,
	protocol string,
	address string,
) (forwarding.Endpoint, error) {
	// Create a cancellable context that we can use to regulate connections.
	dialingContext, dialingCancel := context.WithCancel(context.Background())

	// Create the dialer (unless we're targeting a Windows named pipe).
	var dialer *net.Dialer
	if protocol != "npipe" {
		dialer = &net.Dialer{}
	}

	// Create the endpoint.
	return &dialerEndpoint{
		dialingContext: dialingContext,
		dialingCancel:  dialingCancel,
		dialer:         dialer,
		protocol:       protocol,
		address:        address,
	}, nil
}

// TransportErrors implements forwarding.Endpoint.TransportErrors.
func (e *dialerEndpoint) TransportErrors() <-chan error {
	// Local endpoints don't have a transport that can fail, so we can return an
	// unbuffered empty channel that will never be populated.
	return make(chan error)
}

// Open implements forwarding.Endpoint.Open.
func (e *dialerEndpoint) Open() (net.Conn, error) {
	// If we're dealing with a Windows named pipe target, then perform dialing
	// using the platform-specific dialing function.
	if e.protocol == "npipe" {
		return dialWindowsNamedPipe(e.dialingContext, e.address)
	}

	// For all other protocols (i.e. TCP and Unix domain sockets), use the
	// standard dialer.
	return e.dialer.DialContext(e.dialingContext, e.protocol, e.address)
}

// Shutdown implements forwarding.Endpoint.Shutdown.
func (e *dialerEndpoint) Shutdown() error {
	// Cancel the dialing context to unblock any dialing operations.
	e.dialingCancel()

	// Success.
	return nil
}
