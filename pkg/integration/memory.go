package integration

import (
	"net"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/session"
	urlpkg "github.com/havoc-io/mutagen/pkg/url"
)

const (
	// inMemoryProtocol is a fake protocol used to perform integration tests
	// over an in-memory setup of the agent package's endpoint client/server
	// architecture.
	inMemoryProtocol urlpkg.Protocol = -1
)

// inMemoryProtocolHandler is a protocol handler used to perform integration
// tests over an in-memory setup of the agent package's endpoint client/server
// architecture.
type inMemoryProtocolHandler struct{}

// Dial starts an endpoint server in a background Goroutine and creates an
// endpoint client connected to the server via an in-memory connection.
func (h *inMemoryProtocolHandler) Dial(
	url *urlpkg.URL,
	session string,
	version session.Version,
	configuration *session.Configuration,
	alpha bool,
	prompter string,
) (session.Endpoint, error) {
	// Verify that the URL is of the correct protocol.
	if url.Protocol != inMemoryProtocol {
		panic("non-in-memory URL dispatched to in-memory URL handler")
	}

	// Create an in-memory network connection.
	clientConnection, serverConnection := net.Pipe()

	// Server the endpoint in a background Goroutine. This will terminate once
	// the client connection is closed.
	go agent.ServeEndpoint(serverConnection)

	// Create a client for this endpoint.
	endpoint, err := agent.NewEndpointClient(
		clientConnection,
		session,
		version,
		url.Path,
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
	session.ProtocolHandlers[inMemoryProtocol] = &inMemoryProtocolHandler{}
}
