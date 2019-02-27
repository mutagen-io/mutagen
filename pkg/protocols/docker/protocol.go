package docker

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/session"
	urlpkg "github.com/havoc-io/mutagen/pkg/url"
)

// protocolHandler implements the session.ProtocolHandler interface for
// connecting to remote endpoints inside Docker containers. It uses the agent
// infrastructure over a docker copy/exec transport.
type protocolHandler struct{}

// Dial connects to a Docker endpoint.
func (h *protocolHandler) Dial(
	url *urlpkg.URL,
	prompter,
	session string,
	version session.Version,
	configuration *session.Configuration,
	alpha bool,
) (session.Endpoint, error) {
	// Verify that the URL is of the correct protocol.
	if url.Protocol != urlpkg.Protocol_Docker {
		panic("non-Docker URL dispatched to Docker protocol handler")
	}

	// Create a transport for the agent to use.
	transport, err := newTransport(url, prompter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create Docker transport")
	}

	// Dial using the agent package with a Docker transport.
	return agent.Dial(transport, prompter, url.Path, session, version, configuration, alpha)
}

func init() {
	// Register the Docker protocol handler with the session package.
	session.ProtocolHandlers[urlpkg.Protocol_Docker] = &protocolHandler{}
}
