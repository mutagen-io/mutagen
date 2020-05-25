package ssh

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/agent/transports/ssh"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/remote"
	"github.com/mutagen-io/mutagen/pkg/logging"
	urlpkg "github.com/mutagen-io/mutagen/pkg/url"
	forwardingurlpkg "github.com/mutagen-io/mutagen/pkg/url/forwarding"
)

// protocolHandler implements the forwarding.ProtocolHandler interface for
// connecting to remote endpoints over SSH. It uses the agent infrastructure
// over an SSH transport.
type protocolHandler struct{}

// dialResult provides asynchronous agent dialing results.
type dialResult struct {
	// connection is the connection returned by agent dialing.
	connection net.Conn
	// error is the error returned by agent dialing.
	error error
}

// Connect connects to an SSH endpoint.
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
	} else if url.Protocol != urlpkg.Protocol_SSH {
		panic("non-SSH URL dispatched to SSH protocol handler")
	}

	// Ensure that no environment variables or parameters are specified. These
	// are neither expected nor supported for SSH URLs.
	if len(url.Environment) > 0 {
		return nil, errors.New("SSH URL contains environment variables")
	} else if len(url.Parameters) > 0 {
		return nil, errors.New("SSH URL contains internal parameters")
	}

	// Parse the target specification from the URL's Path component.
	protocol, address, err := forwardingurlpkg.Parse(url.Path)
	if err != nil {
		return nil, fmt.Errorf("unable to parse target specification: %w", err)
	}

	// Create an SSH agent transport.
	transport, err := ssh.NewTransport(url.User, url.Host, uint16(url.Port), prompter)
	if err != nil {
		return nil, fmt.Errorf("unable to create SSH transport: %w", err)
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
	// Register the SSH protocol handler with the forwarding package.
	forwarding.ProtocolHandlers[urlpkg.Protocol_SSH] = &protocolHandler{}
}
