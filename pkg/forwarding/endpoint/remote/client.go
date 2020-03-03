package remote

import (
	"net"

	"github.com/pkg/errors"

	"github.com/hashicorp/yamux"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/remote/internal/closewrite"
	"github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/remote/internal/monitor"
)

// client is a client for a remote forwarding.Endpoint and implements
// forwarding.Endpoint itself.
type client struct {
	// transportErrors is the transport error channel.
	transportErrors <-chan error
	// multiplexer is the underlying multiplexer.
	multiplexer *yamux.Session
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
	// Monitor the connection for read and write errors. The multiplexer we'll
	// create has an internal read loop that will trigger an error if the
	// transport fails, allowing us to monitor for transport errors without
	// implementing our own heartbeat mechanism.
	connection, transportErrors := monitor.Enable(connection)

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

	// Open the initialization stream. We don't need to close this stream on
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
		transportErrors: transportErrors,
		multiplexer:     multiplexer,
		listener:        source,
	}, nil
}

// TransportErrors implements forwarding.Endpoint.TransportErrors.
func (e *client) TransportErrors() <-chan error {
	return e.transportErrors
}

// Open implements forwarding.Endpoint.Open.
func (c *client) Open() (connection net.Conn, err error) {
	// Perform the appropriate opening operation.
	if c.listener {
		connection, err = c.multiplexer.Accept()
	} else {
		connection, err = c.multiplexer.Open()
	}

	// Check for errors.
	if err != nil {
		return
	}

	// Wrap the connection to enable write closure since yamux doesn't support
	// it natively.
	connection = closewrite.Enable(connection)

	// Done.
	return
}

// Shutdown implements forwarding.Endpoint.Shutdown.
func (c *client) Shutdown() error {
	return c.multiplexer.Close()
}
