package local

import (
	"net"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/forwarding"
)

// listenerEndpoint implements forwarding.Endpoint for listener endpoints.
type listenerEndpoint struct {
	// listener is the underlying listener.
	listener net.Listener
}

// NewListenerEndpoint creates a new forwarding.Endpoint that behaves as a
// listener.
func NewListenerEndpoint(protocol, address string) (forwarding.Endpoint, error) {
	// Create the underlying listener.
	listener, err := net.Listen(protocol, address)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create listener")
	}

	// Create the endpoint.
	return &listenerEndpoint{
		listener: listener,
	}, nil
}

// Open implements forwarding.Endpoint.Open.
func (e *listenerEndpoint) Open() (net.Conn, error) {
	return e.listener.Accept()
}

// Shutdown implements forwarding.Endpoint.Shutdown.
func (e *listenerEndpoint) Shutdown() error {
	return e.listener.Close()
}
