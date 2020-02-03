package netpipe

import (
	"context"
	"net"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/remote"
	"github.com/mutagen-io/mutagen/pkg/logging"
	urlpkg "github.com/mutagen-io/mutagen/pkg/url"
	forwardingurl "github.com/mutagen-io/mutagen/pkg/url/forwarding"
)

// forwardingProtocolHandler implements the forwarding.ProtocolHandler interface
// for connecting to "remote" endpoints that actually exist in memory via an
// in-memory pipe.
type forwardingProtocolHandler struct{}

// Dial starts an endpoint server in a background Goroutine and creates an
// endpoint client connected to the server via an in-memory connection.
func (h *forwardingProtocolHandler) Connect(
	_ context.Context,
	logger *logging.Logger,
	url *urlpkg.URL,
	prompter string,
	session string,
	version forwarding.Version,
	configuration *forwarding.Configuration,
	source bool,
) (forwarding.Endpoint, error) {
	// Verify that the URL is of the correct kind and protocol.
	if url.Kind != urlpkg.Kind_Forwarding {
		panic("non-forwarding URL dispatched to forwarding protocol handler")
	} else if url.Protocol != Protocol_Netpipe {
		panic("non-netpipe URL dispatched to netpipe protocol handler")
	}

	// Parse the target specification from the URL's Path component.
	protocol, address, err := forwardingurl.Parse(url.Path)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse target specification")
	}

	// Create an in-memory network connection.
	clientConnection, serverConnection := net.Pipe()

	// Server the endpoint in a background Goroutine. This will terminate once
	// the client connection is closed.
	go remote.ServeEndpoint(logger.Sublogger("remote"), serverConnection)

	// Create a client for this endpoint.
	endpoint, err := remote.NewEndpoint(
		clientConnection,
		version,
		configuration,
		protocol,
		address,
		source,
	)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create in-memory endpoint client")
	}

	// Success.
	return endpoint, nil
}

func init() {
	// Register the netpipe protocol handler with the forwarding package.
	forwarding.ProtocolHandlers[Protocol_Netpipe] = &forwardingProtocolHandler{}
}
