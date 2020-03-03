package tunnel

import (
	"context"
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/endpoint/remote"
	"github.com/mutagen-io/mutagen/pkg/tunneling"
	urlpkg "github.com/mutagen-io/mutagen/pkg/url"
)

// protocolHandler implements the synchronization.ProtocolHandler interface for
// connecting to remote endpoints over tunnels. It uses an underlying tunnel
// manager to create connections.
type protocolHandler struct {
	// manager is the underlying tunnel manager.
	manager *tunneling.Manager
}

// RegisterManager registers the specified tunnel manager as a protocol handler.
func RegisterManager(manager *tunneling.Manager) {
	synchronization.ProtocolHandlers[urlpkg.Protocol_Tunnel] = &protocolHandler{
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
	version synchronization.Version,
	configuration *synchronization.Configuration,
	alpha bool,
) (synchronization.Endpoint, error) {
	// Verify that the URL is of the correct kind and protocol.
	if url.Kind != urlpkg.Kind_Synchronization {
		panic("non-synchronization URL dispatched to synchronization protocol handler")
	} else if url.Protocol != urlpkg.Protocol_Tunnel {
		panic("non-tunnel URL dispatched to tunnel protocol handler")
	}

	// Dial an agent over the tunnel in endpoint mode.
	connection, err := h.manager.Dial(ctx, url.Host, agent.ModeSynchronizer, prompter)
	if err != nil {
		return nil, fmt.Errorf("unable to dial agent endpoint: %w", err)
	}

	// Create the endpoint client.
	return remote.NewEndpoint(connection, url.Path, session, version, configuration, alpha)
}
