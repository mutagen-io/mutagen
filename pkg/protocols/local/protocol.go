package local

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/session/endpoint/local"
	urlpkg "github.com/havoc-io/mutagen/pkg/url"
)

// protocolHandler implements the session.ProtocolHandler interface for
// connecting to local endpoints.
type protocolHandler struct{}

// Dial connects to a local endpoint.
func (h *protocolHandler) Dial(
	url *urlpkg.URL,
	prompter,
	session string,
	version session.Version,
	configuration *session.Configuration,
	alpha bool,
) (session.Endpoint, error) {
	// Verify that the URL is of the correct protocol.
	if url.Protocol != urlpkg.Protocol_Local {
		panic("non-local URL dispatched to local protocol handler")
	}

	// Create a local endpoint.
	endpoint, err := local.NewEndpoint(url.Path, session, version, configuration, alpha)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create local endpoint")
	}

	// Success.
	return endpoint, nil
}

func init() {
	// Register the local protocol handler with the session package.
	session.ProtocolHandlers[urlpkg.Protocol_Local] = &protocolHandler{}
}
