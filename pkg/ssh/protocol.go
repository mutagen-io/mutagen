package ssh

import (
	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/session"
	urlpkg "github.com/havoc-io/mutagen/pkg/url"
)

// protocolHandler implements the session.ProtocolHandler interface for
// connecting to remote endpoints over SSH. It uses the agent infrastructure
// over an SSH transport.
type protocolHandler struct{}

// Dial connects to an SSH endpoint.
func (h *protocolHandler) Dial(
	url *urlpkg.URL,
	prompter,
	session string,
	version session.Version,
	configuration *session.Configuration,
	alpha bool,
) (session.Endpoint, error) {
	// Verify that the URL is of the correct protocol.
	if url.Protocol != urlpkg.Protocol_SSH {
		panic("non-SSH URL dispatched to SSH protocol handler")
	}

	// Create a transport for the agent to use.
	transport := &transport{url, prompter}

	// Dial using the agent package with an SSH transport.
	return agent.Dial(transport, prompter, url.Path, session, version, configuration, alpha)
}

func init() {
	// Register the SSH protocol handler with the session package.
	session.ProtocolHandlers[urlpkg.Protocol_SSH] = &protocolHandler{}
}
