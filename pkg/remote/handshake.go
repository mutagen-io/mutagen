package remote

import (
	"fmt"
)

// handshakeTransportError indicates a handshake error due to a transport
// failure.
type handshakeTransportError struct {
	// underlying is the underlying error that we know is due to a transport
	// failure during the handshake process.
	underlying error
}

// Error returns a formatted version of the transport error.
func (e *handshakeTransportError) Error() string {
	return fmt.Sprintf("handshake transport error: %v", e.underlying)
}

// IsHandshakeTransportError indicates whether or not an error value is a
// handshake transport error.
func IsHandshakeTransportError(err error) bool {
	_, ok := err.(*handshakeTransportError)
	return ok
}
