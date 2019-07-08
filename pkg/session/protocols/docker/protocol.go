package docker

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/agent/transports/docker"
	"github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/session/endpoint/remote"
	urlpkg "github.com/havoc-io/mutagen/pkg/url"
)

// protocolHandler implements the session.ProtocolHandler interface for
// connecting to remote endpoints inside Docker containers. It uses the agent
// infrastructure over a Docker transport.
type protocolHandler struct{}

// Connect connects to a Docker endpoint.
func (h *protocolHandler) Connect(
	url *urlpkg.URL,
	prompter string,
	session string,
	version session.Version,
	configuration *session.Configuration,
	alpha bool,
	ephemeral bool,
) (session.Endpoint, error) {
	// Verify that the URL is of the correct protocol.
	if url.Protocol != urlpkg.Protocol_Docker {
		panic("non-Docker URL dispatched to Docker protocol handler")
	}

	// Create a Docker agent transport.
	transport, err := docker.NewTransport(url.Hostname, url.Username, url.Environment, prompter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create Docker transport")
	}

	// Dial an agent in endpoint mode.
	connection, err := agent.Dial(transport, agent.ModeEndpoint, prompter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to dial agent endpoint")
	}

	// Create the endpoint client.
	return remote.NewEndpointClient(connection, url.Path, session, version, configuration, alpha, ephemeral)
}

func init() {
	// Register the Docker protocol handler with the session package.
	session.ProtocolHandlers[urlpkg.Protocol_Docker] = &protocolHandler{}
}
