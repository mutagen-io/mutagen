package synchronization

import (
	"context"
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/logging"
	urlpkg "github.com/mutagen-io/mutagen/pkg/url"
)

// ProtocolHandler defines the interface that protocol handlers must support in
// order to connect to endpoints.
type ProtocolHandler interface {
	// Connect connects to an endpoint using the connection parameters in the
	// provided URL and the specified prompter (if any). It then initializes the
	// endpoint using the specified parameters.
	Connect(
		ctx context.Context,
		logger logging.Logger,
		url *urlpkg.URL,
		prompter string,
		session string,
		version Version,
		configuration *Configuration,
		alpha bool,
	) (Endpoint, error)
}

// ProtocolHandlers is a map of registered protocol handlers. It should only be
// modified during init() operations.
var ProtocolHandlers = map[urlpkg.Protocol]ProtocolHandler{}

// connect attempts to establish a connection to an endpoint.
func connect(
	ctx context.Context,
	logger logging.Logger,
	url *urlpkg.URL,
	prompter string,
	session string,
	version Version,
	configuration *Configuration,
	alpha bool,
) (Endpoint, error) {
	// Local the appropriate protocol handler.
	handler, ok := ProtocolHandlers[url.Protocol]
	if !ok {
		return nil, fmt.Errorf("unknown protocol: %s", url.Protocol)
	} else if handler == nil {
		panic("nil protocol handler registered")
	}

	// Dispatch the dialing.
	endpoint, err := handler.Connect(ctx, logger, url, prompter, session, version, configuration, alpha)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to endpoint: %w", err)
	}

	// Success.
	return endpoint, nil
}
