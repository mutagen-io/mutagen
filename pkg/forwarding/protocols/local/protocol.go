package local

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/forwarding"
	"github.com/havoc-io/mutagen/pkg/forwarding/endpoint/local"
	"github.com/havoc-io/mutagen/pkg/logging"
	urlpkg "github.com/havoc-io/mutagen/pkg/url"
	forwardingurlpkg "github.com/havoc-io/mutagen/pkg/url/forwarding"
)

// protocolHandler implements the forwarding.ProtocolHandler interface for
// connecting to local forwarding endpoints.
type protocolHandler struct{}

// Connect implements forwarding.ProtocolHandler.Connect.
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
	} else if url.Protocol != urlpkg.Protocol_Local {
		panic("non-local URL dispatched to local protocol handler")
	}

	// Parse the target specification from the URL's Path component.
	protocol, address, err := forwardingurlpkg.Parse(url.Path)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse target specification")
	}

	// Handle creation based on mode.
	if source {
		return local.NewListenerEndpoint(protocol, address)
	} else {
		return local.NewDialerEndpoint(protocol, address)
	}
}

func init() {
	// Register the local protocol handler with the forwarding package.
	forwarding.ProtocolHandlers[urlpkg.Protocol_Local] = &protocolHandler{}
}
