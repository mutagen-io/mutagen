package remote

import (
	"context"
	"net"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/local"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/multiplexing"
)

// ServeEndpoint creates and serves a remote endpoint on the specified
// connection. It enforces that the provided connection is closed by the time
// this function returns, regardless of failure.
func ServeEndpoint(logger *logging.Logger, connection net.Conn) error {
	// Multiplex the connection and defer closure of the multiplexer.
	multiplexer := multiplexing.Multiplex(connection, true, nil)
	defer multiplexer.Close()

	// Accept the initialization stream. We don't need to close this stream on
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

	// If this is a Unix domain socket endpoint, perform normalization on the
	// socket path.
	address := request.Address
	if request.Protocol == "unix" {
		if a, err := filesystem.Normalize(address); err != nil {
			return errors.Wrap(err, "unable to normalize socket path")
		} else {
			address = a
		}
	}

	// Create the underlying endpoint based on the initialization parameters.
	var endpoint forwarding.Endpoint
	var initializationError error
	if request.Listener {
		endpoint, initializationError = local.NewListenerEndpoint(
			request.Version,
			request.Configuration,
			request.Protocol,
			address,
			false,
		)
	} else {
		endpoint, initializationError = local.NewDialerEndpoint(
			request.Version,
			request.Configuration,
			request.Protocol,
			address,
		)
	}

	// If we successfully created the underlying endpoint, then start a
	// Goroutine couple the lifetime of the underlying endpoint to the lifetime
	// of the multiplexer. This will cause the local endpoint to be shutdown if
	// this function returns or if the multiplexer shuts down due to remote
	// closure or an internal error. This is particularly important for
	// preempting local accept operations.
	if endpoint != nil {
		go func() {
			<-multiplexer.Closed()
			endpoint.Shutdown()
		}()
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
		// Receive the next incoming connection. If this fails, then we should
		// terminate serving because either the local listener has failed or the
		// multiplexer has failed.
		var incoming net.Conn
		if request.Listener {
			if incoming, err = endpoint.Open(); err != nil {
				return errors.Wrap(err, "listener failure")
			}
		} else {
			incoming, err = multiplexer.Accept()
			if err != nil {
				return errors.Wrap(err, "multiplexer failure")
			}
		}

		// Open the corresponding outgoing connection. If the multiplexer fails,
		// then we should terminate serving. If local dialing fails, then we can
		// just close the incoming connection to indicate dialing failure.
		var outgoing net.Conn
		if request.Listener {
			outgoing, err = multiplexer.OpenStream(context.Background())
			if err != nil {
				incoming.Close()
				return errors.Wrap(err, "multiplexer failure")
			}
		} else {
			if outgoing, err = endpoint.Open(); err != nil {
				incoming.Close()
				continue
			}
		}

		// Perform forwarding.
		go forwarding.ForwardAndClose(context.Background(), incoming, outgoing)
	}
}
