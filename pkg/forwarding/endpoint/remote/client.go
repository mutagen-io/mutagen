package remote

import (
	"context"
	"net"

	"github.com/pkg/errors"

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
// specified connection with the specified metadata. If this function fails,
// then the provided connection will be closed. Once the endpoint has been
// established, the underlying connection is owned by that endpoint and will be
// closed when the endpoint is shut down.
func NewEndpoint(
	connection net.Conn,
	version forwarding.Version,
	configuration *forwarding.Configuration,
	protocol string,
	address string,
	source bool,
) (forwarding.Endpoint, error) {
	// Multiplex the connection.
	multiplexer := multiplexing.Multiplex(connection, false, nil)

	// Defer closure of the multiplexer in the event that we're unsuccessful.
	var successful bool
	defer func() {
		if !successful {
			multiplexer.Close()
		}
	}()

	// Open the initialization stream. We don't need to close this stream on
	// failure since closing the multiplexer will implicitly close the stream.
	stream, err := multiplexer.OpenStream(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "unable to open initialization stream")
	}

	// Create and send the initialization request.
	request := &InitializeForwardingRequest{
		Version:       version,
		Configuration: configuration,
		Protocol:      protocol,
		Address:       address,
		Listener:      source,
	}
	if err := encoding.NewProtobufEncoder(stream).Encode(request); err != nil {
		return nil, errors.Wrap(err, "unable to send initialization request")
	}

	// Receive the initialization response, ensure that it's valid, and check
	// for initialization errors.
	response := &InitializeForwardingResponse{}
	if err := encoding.NewProtobufDecoder(stream).Decode(response); err != nil {
		return nil, errors.Wrap(err, "unable to receive initialization response")
	} else if err = response.ensureValid(); err != nil {
		return nil, errors.Wrap(err, "invalid initialization response received")
	} else if response.Error != "" {
		return nil, errors.Wrap(errors.New(response.Error), "remote initialization failure")
	}

	// Close the initialization stream.
	if err := stream.Close(); err != nil {
		return nil, errors.Wrap(err, "unable to close initialization stream")
	}

	// Mark initialization as successful.
	successful = true

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
