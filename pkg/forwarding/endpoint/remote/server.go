package remote

import (
	"io"
	"net"

	"github.com/pkg/errors"

	"github.com/hashicorp/yamux"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/local"
	"github.com/mutagen-io/mutagen/pkg/logging"
)

// ServeEndpoint creates and serves a remote endpoint server on the specified
// connection. It enforces that the provided connection is closed by the time
// this function returns, regardless of failure.
func ServeEndpoint(logger *logging.Logger, connection net.Conn) error {
	// Wrap the connection in a multiplexer. This constructor won't close the
	// underlying connnection on error, so we need to do that manually if an
	// error occurs.
	multiplexer, err := yamux.Server(connection, nil)
	if err != nil {
		connection.Close()
		return errors.Wrap(err, "unable to create multiplexer")
	}

	// Defer closure of the multiplexer (which will close the underlying
	// connection).
	defer multiplexer.Close()

	// Accept the initialization stream. We use a separate stream for this
	// (rather than initializing on the underlying connection) because
	// encoding.ProtobufDecoder uses a buffered reader wrapper and thus we can't
	// rely on the state of the line after using it. Accepting a stream also
	// serves as a reasonable sanity test. We don't need to close this stream on
	// failure since closing the multiplexer will implicitly close the stream.
	stream, err := multiplexer.Accept()
	if err != nil {
		return errors.Wrap(err, "unable to accept initialization stream")
	}

	// Receive the initialization request and ensure that it's valid.
	request := &InitializeForwardingRequest{}
	if err := encoding.NewProtobufDecoder(stream).Decode(request); err != nil {
		return errors.Wrap(err, "unable to receive initialization request")
	} else if err = request.ensureValid(); err != nil {
		return errors.Wrap(err, "invalid initialization request received")
	}

	// Create the underlying endpoint based on the initialization parameters.
	var endpoint forwarding.Endpoint
	var initializationError error
	if request.Listener {
		endpoint, initializationError = local.NewListenerEndpoint(
			request.Version,
			request.Configuration,
			request.Protocol,
			request.Address,
		)
	} else {
		endpoint, initializationError = local.NewDialerEndpoint(
			request.Version,
			request.Configuration,
			request.Protocol,
			request.Address,
		)
	}

	// Send the initialization response, indicating any initialization error
	// that occurred.
	response := &InitializeForwardingResponse{}
	if initializationError != nil {
		response.Error = initializationError.Error()
	}
	if err := encoding.NewProtobufEncoder(stream).Encode(response); err != nil {
		return errors.Wrap(err, "unable to send initialization response")
	}

	// Check for initialization errors.
	if initializationError != nil {
		return errors.Wrap(initializationError, "endpoint initialization failed")
	}

	// Close the initialization stream.
	if err := stream.Close(); err != nil {
		return errors.Wrap(err, "unable to close initialization stream")
	}

	// Receive and forward connections indefinitely.
	for {
		// Receive the next connection. If this fails, then we should terminate
		// serving because either the local listener has failed or the
		// multiplexer has failed.
		var receivedConnection net.Conn
		if request.Listener {
			if receivedConnection, err = endpoint.Open(); err != nil {
				return errors.Wrap(err, "listener failure")
			}
		} else {
			if receivedConnection, err = multiplexer.Accept(); err != nil {
				return errors.Wrap(err, "multiplexer failure")
			}
		}

		// Open the corresponding target connection. If the multiplexer fails,
		// then we should terminate serving. If local dialing fails, then we can
		// just close the accepted connection.
		var outgoingConnection net.Conn
		if request.Listener {
			if outgoingConnection, err = multiplexer.Open(); err != nil {
				return errors.Wrap(err, "multiplexer failure")
			}
		} else {
			if outgoingConnection, err = endpoint.Open(); err != nil {
				receivedConnection.Close()
				continue
			}
		}

		// Perform forwarding.
		go forwardAndClose(receivedConnection, outgoingConnection)
	}
}

// forwardAndClose is a simple utility function designed to perform connection
// forwarding (and closure on failure) in a background Goroutine.
func forwardAndClose(first, second net.Conn) {
	// Forward in background Goroutines and track failure.
	copyErrors := make(chan error, 2)
	go func() {
		_, err := io.Copy(first, second)
		copyErrors <- err
	}()
	go func() {
		_, err := io.Copy(second, first)
		copyErrors <- err
	}()

	// Wait for a copy error.
	<-copyErrors

	// Close both connections.
	first.Close()
	second.Close()
}
