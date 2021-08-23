package remote

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/multiplexing"
)

// client is a client for a remote forwarding.Endpoint and implements
// forwarding.Endpoint itself.
type client struct {
	// transportErrors is the transport error channel.
	transportErrors <-chan error
	// multiplexer is the underlying multiplexer.
	multiplexer *multiplexing.Multiplexer
	// listener indicates whether or not the remote endpoint is operating as a
	// listener.
	listener bool
}

// NewEndpoint creates a new remote forwarding.Endpoint operating over the
// specified stream with the specified metadata. If this function fails, then
// the provided stream will be closed. Once the endpoint has been established,
// the underlying stream is owned by the endpoint and will be closed when the
// endpoint is shut down. The provided stream must unblock read and write
// operations when closed.
func NewEndpoint(
	stream io.ReadWriteCloser,
	version forwarding.Version,
	configuration *forwarding.Configuration,
	protocol string,
	address string,
	source bool,
) (forwarding.Endpoint, error) {
	// Adapt the stream to serve as a multiplexer carrier. This will also give
	// us the buffering functionality we'll need for initialization.
	carrier := multiplexing.NewCarrierFromStream(stream)

	// Defer closure of the carrier in the event that initialization isn't
	// successful. Otherwise, we'll rely on closure of the multiplexer to close
	// the carrier.
	var initializationSuccessful bool
	defer func() {
		if !initializationSuccessful {
			carrier.Close()
		}
	}()

	// Create and send the initialization request.
	request := &InitializeForwardingRequest{
		Version:       version,
		Configuration: configuration,
		Protocol:      protocol,
		Address:       address,
		Listener:      source,
	}
	if err := encoding.EncodeProtobuf(carrier, request); err != nil {
		return nil, fmt.Errorf("unable to send initialization request: %w", err)
	}

	// Receive the initialization response, ensure that it's valid, and check
	// for initialization errors.
	response := &InitializeForwardingResponse{}
	if err := encoding.DecodeProtobuf(carrier, response); err != nil {
		return nil, fmt.Errorf("unable to receive initialization response: %w", err)
	} else if err = response.ensureValid(); err != nil {
		return nil, fmt.Errorf("invalid initialization response received: %w", err)
	} else if response.Error != "" {
		return nil, fmt.Errorf("remote initialization failure: %w", errors.New(response.Error))
	}

	// Mark initialization as successful.
	initializationSuccessful = true

	// Multiplex the carrier.
	multiplexer := multiplexing.Multiplex(carrier, false, nil)

	// Create a channel to monitor for transport errors and a Goroutine to
	// populate it.
	transportErrors := make(chan error, 1)
	go func() {
		<-multiplexer.Closed()
		if err := multiplexer.InternalError(); err != nil {
			transportErrors <- err
		} else {
			transportErrors <- multiplexing.ErrMultiplexerClosed
		}
	}()

	// Success.
	return &client{
		transportErrors: transportErrors,
		multiplexer:     multiplexer,
		listener:        source,
	}, nil
}

// TransportErrors implements forwarding.Endpoint.TransportErrors.
func (c *client) TransportErrors() <-chan error {
	return c.transportErrors
}

// Open implements forwarding.Endpoint.Open.
func (c *client) Open() (net.Conn, error) {
	if c.listener {
		return c.multiplexer.Accept()
	} else {
		stream, err := c.multiplexer.OpenStream(context.Background())
		return stream, err
	}
}

// Shutdown implements forwarding.Endpoint.Shutdown.
func (c *client) Shutdown() error {
	return c.multiplexer.Close()
}
