package local

import (
	"context"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/local"
	"github.com/mutagen-io/mutagen/pkg/logging"
	urlpkg "github.com/mutagen-io/mutagen/pkg/url"
	forwardingurl "github.com/mutagen-io/mutagen/pkg/url/forwarding"
)

// protocolHandler implements the forwarding.ProtocolHandler interface for
// connecting to local forwarding endpoints.
type protocolHandler struct{}

// Connect implements forwarding.ProtocolHandler.Connect.
func (p *protocolHandler) Connect(
	_ context.Context,
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
	} else if url.Protocol != urlpkg.Protocol_Local {
		panic("non-local URL dispatched to local protocol handler")
	}

	// Ensure that no environment variables or parameters are specified. These
	// are neither expected nor supported for local URLs.
	if len(url.Environment) > 0 {
		return nil, errors.New("local URL contains environment variables")
	} else if len(url.Parameters) > 0 {
		return nil, errors.New("local URL contains internal parameters")
	}

	// Parse the target specification from the URL's Path component.
	protocol, address, err := forwardingurl.Parse(url.Path)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse target specification")
	}

	// Handle creation based on mode.
	if source {
		return local.NewListenerEndpoint(version, configuration, protocol, address, true)
	} else {
		return local.NewDialerEndpoint(version, configuration, protocol, address)
	}
}

func init() {
	// Register the local protocol handler with the forwarding package.
	forwarding.ProtocolHandlers[urlpkg.Protocol_Local] = &protocolHandler{}
}
