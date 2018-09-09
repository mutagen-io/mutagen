package integration

import (
	"net"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/remote"
	"github.com/havoc-io/mutagen/pkg/session"
	urlpkg "github.com/havoc-io/mutagen/pkg/url"
)

const (
	// inMemoryProtocol is a fake protocol used to perform integration tests
	// over an in-memory setup of the remote client/server architecture.
	inMemoryProtocol urlpkg.Protocol = -1
)

// protocolHandler implements the session.ProtocolHandler interface for
// connecting to "remote" endpoints that actually exist in memory via an
// in-memory pipe.
type protocolHandler struct{}

// Dial starts an endpoint server in a background Goroutine and creates an
// endpoint client connected to the server via an in-memory connection.
func (h *protocolHandler) Dial(
	url *urlpkg.URL,
	prompter,
	session string,
	version session.Version,
	configuration *session.Configuration,
	alpha bool,
) (session.Endpoint, error) {
	// Verify that the URL is of the correct protocol.
	if url.Protocol != inMemoryProtocol {
		panic("non-in-memory URL dispatched to in-memory protocol handler")
	}

	// Create an in-memory network connection.
	clientConnection, serverConnection := net.Pipe()

	// Server the endpoint in a background Goroutine. This will terminate once
	// the client connection is closed.
	go remote.ServeEndpoint(serverConnection)

	// Create a client for this endpoint.
	endpoint, err := remote.NewEndpointClient(
		clientConnection,
		url.Path,
		session,
		version,
		configuration,
		alpha,
	)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create in-memory endpoint client")
	}

	// Success.
	return endpoint, nil
}

func init() {
	// Register the in-memory protocol handler with the session package.
	session.ProtocolHandlers[inMemoryProtocol] = &protocolHandler{}
}
