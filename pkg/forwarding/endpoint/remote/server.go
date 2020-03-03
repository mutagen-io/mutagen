package remote

import (
	"context"
	"net"

	"github.com/pkg/errors"

	"github.com/hashicorp/yamux"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/local"
	"github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/remote/internal/closewrite"
	"github.com/mutagen-io/mutagen/pkg/logging"
)

// ServeEndpoint creates and serves a remote endpoint on the specified
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
		// multiplexer has failed. We wrap multiplexer connections to enable
		// write closure since yamux doesn't support it natively.
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
			incoming = closewrite.Enable(incoming)
		}

		// Open the corresponding outgoing connection. If the multiplexer fails,
		// then we should terminate serving. If local dialing fails, then we can
		// just close the incoming connection. We wrap multiplexer connections
		// to enable write closure since yamux doesn't support it natively.
		var outgoing net.Conn
		if request.Listener {
			outgoing, err = multiplexer.Open()
			if err != nil {
				incoming.Close()
				return errors.Wrap(err, "multiplexer failure")
			}
			outgoing = closewrite.Enable(outgoing)
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
