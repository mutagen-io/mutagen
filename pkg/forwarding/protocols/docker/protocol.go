package docker

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/agent/transports/docker"
	"github.com/havoc-io/mutagen/pkg/forwarding"
	"github.com/havoc-io/mutagen/pkg/forwarding/endpoint/remote"
	"github.com/havoc-io/mutagen/pkg/logging"
	urlpkg "github.com/havoc-io/mutagen/pkg/url"
	forwardingurlpkg "github.com/havoc-io/mutagen/pkg/url/forwarding"
)

// protocolHandler implements the forwarding.ProtocolHandler interface for
// connecting to remote forwarding endpoints inside Docker containers. It uses
// the agent infrastructure over a Docker transport.
type protocolHandler struct{}

// Connect connects to a Docker endpoint.
func (p *protocolHandler) Connect(
	logger *logging.Logger,
	url *urlpkg.URL, prompter string,
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
		return nil, errors.Wrap(err, "unable to parse target specification")
	}

	// Create a Docker agent transport.
	transport, err := docker.NewTransport(url.Host, url.User, url.Environment, prompter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create Docker transport")
	}

	// Dial an agent in forwarding mode.
	connection, err := agent.Dial(logger, transport, agent.ModeForwarder, prompter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to dial agent endpoint")
	}

	// Create the endpoint.
	return remote.NewEndpoint(connection, version, configuration, protocol, address, source)
}

func init() {
	// Register the Docker protocol handler with the forwarding package.
	forwarding.ProtocolHandlers[urlpkg.Protocol_Docker] = &protocolHandler{}
}
