package tunnel

import (
	"context"
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/remote"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/tunneling"
	urlpkg "github.com/mutagen-io/mutagen/pkg/url"
	forwardingurlpkg "github.com/mutagen-io/mutagen/pkg/url/forwarding"
)

// protocolHandler implements the forwarding.ProtocolHandler interface for
// connecting to remote endpoints over tunnels. It uses an underlying tunnel
// manager to create connections.
type protocolHandler struct {
	// manager is the underlying tunnel manager.
	manager *tunneling.Manager
}

// RegisterManager registers the specified tunnel manager as a protocol handler.
func RegisterManager(manager *tunneling.Manager) {
	forwarding.ProtocolHandlers[urlpkg.Protocol_Tunnel] = &protocolHandler{
		manager: manager,
	}
}

// Connect connects to an tunnel-based endpoint.
func (h *protocolHandler) Connect(
	ctx context.Context,
	_ *logging.Logger,
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
	} else if url.Protocol != urlpkg.Protocol_Tunnel {
		panic("non-tunnel URL dispatched to tunnel protocol handler")
	}

	// Parse the target specification from the URL's Path component.
	protocol, address, err := forwardingurlpkg.Parse(url.Path)
	if err != nil {
		return nil, fmt.Errorf("unable to parse target specification: %w", err)
	}

	// Dial an agent over the tunnel in endpoint mode.
	connection, err := h.manager.Dial(ctx, url.Host, agent.ModeForwarder, prompter)
	if err != nil {
		return nil, fmt.Errorf("unable to dial agent endpoint: %w", err)
	}

	// Create the endpoint.
	return remote.NewEndpoint(connection, version, configuration, protocol, address, source)
}
