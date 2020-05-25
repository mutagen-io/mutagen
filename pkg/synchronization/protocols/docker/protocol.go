package docker

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/agent/transports/docker"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/endpoint/remote"
	urlpkg "github.com/mutagen-io/mutagen/pkg/url"
)

// protocolHandler implements the synchronization.ProtocolHandler interface for
// connecting to remote endpoints inside Docker containers. It uses the agent
// infrastructure over a Docker transport.
type protocolHandler struct{}

// dialResult provides asynchronous agent dialing results.
type dialResult struct {
	// connection is the connection returned by agent dialing.
	connection net.Conn
	// error is the error returned by agent dialing.
	error error
}

// Connect connects to a Docker endpoint.
func (h *protocolHandler) Connect(
	ctx context.Context,
	logger *logging.Logger,
	url *urlpkg.URL,
	prompter string,
	session string,
	version synchronization.Version,
	configuration *synchronization.Configuration,
	alpha bool,
) (synchronization.Endpoint, error) {
	// Verify that the URL is of the correct kind and protocol.
	if url.Kind != urlpkg.Kind_Synchronization {
		panic("non-synchronization URL dispatched to synchronization protocol handler")
	} else if url.Protocol != urlpkg.Protocol_Docker {
		panic("non-Docker URL dispatched to Docker protocol handler")
	}

	// Create a Docker agent transport.
	transport, err := docker.NewTransport(url.Host, url.User, url.Environment, url.Parameters, prompter)
	if err != nil {
		return nil, fmt.Errorf("unable to create Docker transport: %w", err)
	}

	// Create a channel to deliver the dialing result.
	results := make(chan dialResult)

	// Perform dialing in a background Goroutine so that we can monitor for
	// cancellation.
	go func() {
		// Perform the dialing operation.
		connection, err := agent.Dial(logger, transport, agent.ModeSynchronizer, prompter)

		// Transmit the result or, if cancelled, close the connection.
		select {
		case results <- dialResult{connection, err}:
		case <-ctx.Done():
			if connection != nil {
				connection.Close()
			}
		}
	}()

	// Wait for dialing results or cancellation.
	var connection net.Conn
	select {
	case result := <-results:
		if result.error != nil {
			return nil, fmt.Errorf("unable to dial agent endpoint: %w", result.error)
		}
		connection = result.connection
	case <-ctx.Done():
		return nil, errors.New("connect operation cancelled")
	}

	// Create the endpoint client.
	return remote.NewEndpoint(connection, url.Path, session, version, configuration, alpha)
}

func init() {
	// Register the Docker protocol handler with the synchronization package.
	synchronization.ProtocolHandlers[urlpkg.Protocol_Docker] = &protocolHandler{}
}
