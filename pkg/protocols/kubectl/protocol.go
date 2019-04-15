package kubectl

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/session"
	urlpkg "github.com/havoc-io/mutagen/pkg/url"
)

// protocolHandler implements the session.ProtocolHandler interface for
// connecting to remote endpoints inside Kubectl containers. It uses the agent
// infrastructure over a kubectl copy/exec transport.
type protocolHandler struct{}

// Dial connects to a Kubectl endpoint.
func (h *protocolHandler) Dial(
	url *urlpkg.URL,
	prompter,
	session string,
	version session.Version,
	configuration *session.Configuration,
	alpha bool,
) (session.Endpoint, error) {
	// Verify that the URL is of the correct protocol.
	if url.Protocol != urlpkg.Protocol_Kubectl {
		panic("non-Kubectl URL dispatched to Kubectl protocol handler")
	}

	// Create a transport for the agent to use.
	transport, err := newTransport(url, prompter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create Kubectl transport")
	}

	// Dial using the agent package with a Kubectl transport.
	return agent.Dial(transport, prompter, url.Path, session, version, configuration, alpha)
}

func init() {
	// Register the Kubectl protocol handler with the session package.
	session.ProtocolHandlers[urlpkg.Protocol_Kubectl] = &protocolHandler{}
}
