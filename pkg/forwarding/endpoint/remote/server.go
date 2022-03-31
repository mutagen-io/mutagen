package remote

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/local"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/multiplexing"
)

// initializeEndpoint initializes the underlying endpoint based on the provided
// initialization request.
func initializeEndpoint(logger *logging.Logger, request *InitializeForwardingRequest) (forwarding.Endpoint, error) {
	// If this is a Unix domain socket endpoint, perform normalization on the
	// socket path.
	address := request.Address
	if request.Protocol == "unix" {
		if a, err := filesystem.Normalize(address); err != nil {
			return nil, fmt.Errorf("unable to normalize socket path: %w", err)
		} else {
			address = a
		}
	}

	// Create the underlying endpoint based on the initialization parameters.
	if request.Listener {
		return local.NewListenerEndpoint(
			logger,
			request.Version,
			request.Configuration,
			request.Protocol,
			address,
			false,
		)
	} else {
		return local.NewDialerEndpoint(
			logger,
			request.Version,
			request.Configuration,
			request.Protocol,
			address,
		)
	}
}

// ServeEndpoint creates and serves a remote endpoint on the specified stream.
// It enforces that the provided stream is closed by the time this function
// returns, regardless of failure. The provided stream must unblock read and
// write operations when closed.
func ServeEndpoint(logger *logging.Logger, stream io.ReadWriteCloser) error {
	// Adapt the connection to serve as a multiplexer carrier. This will also
	// give us the buffering functionality we'll need for initialization.
	carrier := multiplexing.NewCarrierFromStream(stream)

	// Defer closure of the carrier in the event that initialization isn't
	// successful. Otherwise, we'll rely on closure of the multiplexer to close
	// the carrier.
	var initializationError error
	defer func() {
		if initializationError != nil {
			carrier.Close()
		}
	}()

	// Receive the initialization request, ensure that it's valid, and perform
	// initialization.
	request := &InitializeForwardingRequest{}
	var underlying forwarding.Endpoint
	if err := encoding.DecodeProtobuf(carrier, request); err != nil {
		initializationError = fmt.Errorf("unable to receive initialization request: %w", err)
	} else if err = request.ensureValid(); err != nil {
		initializationError = fmt.Errorf("invalid initialization request received: %w", err)
	} else {
		underlying, initializationError = initializeEndpoint(logger, request)
	}

	// Send the initialization response, indicating any initialization error
	// that occurred.
	response := &InitializeForwardingResponse{}
	if initializationError != nil {
		response.Error = initializationError.Error()
	}
	if err := encoding.EncodeProtobuf(carrier, response); err != nil {
		return fmt.Errorf("unable to send initialization response: %w", err)
	}

	// If initialization failed, then bail.
	if initializationError != nil {
		return fmt.Errorf("endpoint initialization failed: %w", initializationError)
	}

	// Multiplex the carrier and defer closure of the multiplexer.
	multiplexer := multiplexing.Multiplex(carrier, true, nil)
	defer multiplexer.Close()

	// Start a Goroutine couple the lifetime of the underlying endpoint to the
	// lifetime of the multiplexer. This will cause the underlying endpoint to
	// be shut down if this function returns or if the multiplexer shuts down
	// due to remote closure or an internal error. This is particularly
	// important for preempting local accept operations.
	go func() {
		<-multiplexer.Closed()
		underlying.Shutdown()
	}()

	// Receive and forward connections indefinitely.
	for {
		// Receive the next incoming connection. If this fails, then we should
		// terminate serving because either the local listener has failed or the
		// multiplexer has failed.
		var incoming net.Conn
		var err error
		if request.Listener {
			if incoming, err = underlying.Open(); err != nil {
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
			if outgoing, err = underlying.Open(); err != nil {
				incoming.Close()
				continue
			}
		}

		// Perform forwarding.
		go forwarding.ForwardAndClose(context.Background(), incoming, outgoing)
	}
}
