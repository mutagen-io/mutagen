package forwarding

import (
	"net"
)

// Endpoint is a generic network connectivity interface that can represent both
// listening or dialing. None of its methods should be considered safe for
// concurrent invocation except Shutdown.
type Endpoint interface {
	// TransportErrors returns a channel that will be populated if an error
	// occurs on the underlying transport. This is necessary for forwarding
	// endpoints because (unlike synchronization endpoints) there's no
	// simultaneous polling of both endpoints that will detect connection
	// failure. By monitoring for transport errors separately, the forwarding
	// loop can be cancelled immediately (instead of waiting for a dial
	// operation to fail once the next connection is accepted). The endpoint
	// should make no assumptions about whether this method will be called or
	// whether the resulting channel will be read from. Callers should make no
	// assumptions about whether or not the resulting channel will be populated.
	TransportErrors() <-chan error

	// Open should open a network connection for the endpoint. For listener
	// (source) endpoints, this function should block until an incoming
	// connection arrives. For dialer (destination) endpoints, this function
	// should dial the underlying target.
	Open() (net.Conn, error)

	// Shutdown shuts down the endpoint. This function should unblock any
	// pending Open call.
	Shutdown() error
}
