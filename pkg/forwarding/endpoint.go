package forwarding

import (
	"net"
)

// Endpoint is a generic network connectivity interface that can represent both
// listening or dialing. In general, it is not safe for concurrent calls. It is,
// however, required that Shutdown be callable concurrently with other methods.
type Endpoint interface {
	// Open should open a net connection for the endpoint. For listener (source)
	// endpoints, this function should block until an incoming connection
	// arrives. For dialer (destination) endpoints, this function should dial
	// the underlying target.
	Open() (net.Conn, error)
	// Shutdown shuts down the endpoint. This function should unblock any
	// pending Open call.
	Shutdown() error
}
