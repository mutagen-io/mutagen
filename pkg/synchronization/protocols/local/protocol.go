package local

import (
	"context"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/endpoint/local"
	urlpkg "github.com/mutagen-io/mutagen/pkg/url"
)

// protocolHandler implements the synchronization.ProtocolHandler interface for
// connecting to local endpoints.
type protocolHandler struct{}

// Connect connects to a local endpoint.
func (h *protocolHandler) Connect(
	_ context.Context,
	logger *logging.Logger,
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

	// Create a local endpoint.
	endpoint, err := local.NewEndpoint(logger, url.Path, session, version, configuration, alpha)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create local endpoint")
	}

	// Success.
	return endpoint, nil
}

func init() {
	// Register the local protocol handler with the synchronization package.
	synchronization.ProtocolHandlers[urlpkg.Protocol_Local] = &protocolHandler{}
}
