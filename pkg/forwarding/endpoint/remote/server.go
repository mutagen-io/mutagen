package remote

import (
	"context"
	"fmt"
	"net"

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
		return fmt.Errorf("unable to accept initialization stream: %w", err)
	}

	// Receive the initialization request and ensure that it's valid.
	request := &InitializeForwardingRequest{}
	if err := encoding.NewProtobufDecoder(stream).Decode(request); err != nil {
		return fmt.Errorf("unable to receive initialization request: %w", err)
	} else if err = request.ensureValid(); err != nil {
		return fmt.Errorf("invalid initialization request received: %w", err)
	}

	// If this is a Unix domain socket endpoint, perform normalization on the
	// socket path.
	address := request.Address
	if request.Protocol == "unix" {
		if a, err := filesystem.Normalize(address); err != nil {
			return fmt.Errorf("unable to normalize socket path: %w", err)
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
		return fmt.Errorf("unable to send initialization response: %w", err)
	}

	// Check for initialization errors.
	if initializationError != nil {
		return fmt.Errorf("endpoint initialization failed: %w", initializationError)
	}

	// Close the initialization stream.
	if err := stream.Close(); err != nil {
		return fmt.Errorf("unable to close initialization stream: %w", err)
	}

	// Receive and forward connections indefinitely.
	for {
		// Receive the next incoming connection. If this fails, then we should
		// terminate serving because either the local listener has failed or the
		// multiplexer has failed.
		var incoming net.Conn
		if request.Listener {
			if incoming, err = endpoint.Open(); err != nil {
				return fmt.Errorf("listener failure: %w", err)
			}
		} else {
			incoming, err = multiplexer.Accept()
			if err != nil {
				return fmt.Errorf("multiplexer failure: %w", err)
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
				return fmt.Errorf("multiplexer failure: %w", err)
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
