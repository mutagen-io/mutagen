package docker

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/agent/transports/docker"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/remote"
	"github.com/mutagen-io/mutagen/pkg/logging"
	urlpkg "github.com/mutagen-io/mutagen/pkg/url"
	forwardingurlpkg "github.com/mutagen-io/mutagen/pkg/url/forwarding"
)

// protocolHandler implements the forwarding.ProtocolHandler interface for
// connecting to remote forwarding endpoints inside Docker containers. It uses
// the agent infrastructure over a Docker transport.
type protocolHandler struct{}

// dialResult provides asynchronous agent dialing results.
type dialResult struct {
	// connection is the connection returned by agent dialing.
	connection net.Conn
	// error is the error returned by agent dialing.
	error error
}

// Connect connects to a Docker endpoint.
func (p *protocolHandler) Connect(
	ctx context.Context,
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
	} else if url.Protocol != urlpkg.Protocol_Docker {
		panic("non-Docker URL dispatched to Docker protocol handler")
	}

	// Parse the target specification from the URL's Path component.
	protocol, address, err := forwardingurlpkg.Parse(url.Path)
	if err != nil {
		return nil, fmt.Errorf("unable to parse target specification: %w", err)
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
		connection, err := agent.Dial(logger, transport, agent.ModeForwarder, prompter)

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

	// Create the endpoint.
	return remote.NewEndpoint(connection, version, configuration, protocol, address, source)
}

func init() {
	// Register the Docker protocol handler with the forwarding package.
	forwarding.ProtocolHandlers[urlpkg.Protocol_Docker] = &protocolHandler{}
}
