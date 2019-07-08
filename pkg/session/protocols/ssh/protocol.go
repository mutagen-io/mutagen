package ssh

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/agent/transports/ssh"
	"github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/session/endpoint/remote"
	urlpkg "github.com/havoc-io/mutagen/pkg/url"
)

// protocolHandler implements the session.ProtocolHandler interface for
// connecting to remote endpoints over SSH. It uses the agent infrastructure
// over an SSH transport.
type protocolHandler struct{}

// Dial connects to an SSH endpoint.
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
	if url.Protocol != urlpkg.Protocol_SSH {
		panic("non-SSH URL dispatched to SSH protocol handler")
	}

	// Create an SSH agent transport.
	transport, err := ssh.NewTransport(url.Username, url.Hostname, uint16(url.Port), prompter)
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
	// Register the SSH protocol handler with the session package.
	session.ProtocolHandlers[urlpkg.Protocol_SSH] = &protocolHandler{}
}
