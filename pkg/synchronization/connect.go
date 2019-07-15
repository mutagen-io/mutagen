package synchronization

import (
	"context"

	"github.com/pkg/errors"

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
		logger *logging.Logger,
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
	logger *logging.Logger,
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
		return nil, errors.Errorf("unknown protocol: %s", url.Protocol)
	} else if handler == nil {
		panic("nil protocol handler registered")
	}

	// Dispatch the dialing.
	endpoint, err := handler.Connect(logger, url, prompter, session, version, configuration, alpha)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to endpoint")
	}

	// Success.
	return endpoint, nil
}

// asyncConnectResult provides asynchronous connection results.
type asyncConnectResult struct {
	// endpoint is the endpoint returned by connect.
	endpoint Endpoint
	// error is the error returned by connect.
	error error
}

// reconnect is a version of connect that accepts a context for cancellation. It
// is only designed for auto-reconnection purposes, so it does not accept a
// prompter.
func reconnect(
	ctx context.Context,
	logger *logging.Logger,
	url *urlpkg.URL,
	session string,
	version Version,
	configuration *Configuration,
	alpha bool,
) (Endpoint, error) {
	// Create a channel to deliver the connection result.
	results := make(chan asyncConnectResult)

	// Start a connection operation in the background.
	go func() {
		// Perform the connection.
		endpoint, err := connect(logger, url, "", session, version, configuration, alpha)

		// If we can't transmit the resulting endpoint, shut it down.
		select {
		case <-ctx.Done():
			if endpoint != nil {
				endpoint.Shutdown()
			}
		case results <- asyncConnectResult{endpoint, err}:
		}
	}()

	// Wait for context cancellation or results.
	select {
	case <-ctx.Done():
		return nil, errors.New("reconnect cancelled")
	case result := <-results:
		return result.endpoint, result.error
	}
}
