package remote

import (
	"net"

	"github.com/pkg/errors"

	"github.com/hashicorp/yamux"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
)

// client is a client for a remote forwarding.Endpoint and implements
// forwarding.Endpoint itself.
type client struct {
	// multiplexer is the underlying multiplexer.
	multiplexer *yamux.Session
	// listener indicates whether or not the remote endpoint is operating as a
	// listener.
	listener bool
}

// NewEndpoint creates a new forwarding.Endpoint object that operates over a
// multiplexed connection. If this function fails, then the provided connection
// will be closed. Once the endpoint has been established, the underlying
// connection is owned by that endpoint and will be closed when the endpoint is
// shut down.
func NewEndpoint(
	connection net.Conn,
	version forwarding.Version,
	configuration *forwarding.Configuration,
	protocol string,
	address string,
	source bool,
) (forwarding.Endpoint, error) {
	// Wrap the connection in a multiplexer. This constructor won't close the
	// underlying connnection on error, so we need to do that manually if an
	// error occurs.
	multiplexer, err := yamux.Client(connection, nil)
	if err != nil {
		connection.Close()
		return nil, errors.Wrap(err, "unable to create multiplexer")
	}

	// Defer closure of the multiplexer in the event that we're unsuccessful.
	var successful bool
	defer func() {
		if !successful {
			multiplexer.Close()
		}
	}()

	// Open the initialization stream. We use a separate stream for this (rather
	// than initializing on the underlying connection) because
	// encoding.ProtobufDecoder uses a buffered reader wrapper and thus we can't
	// rely on the state of the line after using it. Opening a stream also
	// serves as a reasonable sanity test. We don't need to close this stream on
	// failure since closing the multiplexer will implicitly close the stream.
	stream, err := multiplexer.Open()
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

	// Success.
	successful = true
	return &client{
		multiplexer: multiplexer,
		listener:    source,
	}, nil
}

// Open implements forwarding.Endpoint.Open.
func (c *client) Open() (net.Conn, error) {
	if c.listener {
		return c.multiplexer.Accept()
	} else {
		return c.multiplexer.Open()
	}
}

// Shutdown implements forwarding.Endpoint.Shutdown.
func (c *client) Shutdown() error {
	return c.multiplexer.Close()
}
