package session

import (
	"context"

	"github.com/pkg/errors"

	urlpkg "github.com/havoc-io/mutagen/pkg/url"
)

// ProtocolHandler defines the interface that protocol handlers must support in
// order to connect to endpoints.
type ProtocolHandler interface {
	// Dial connects to the endpoint at the specified URL with the specified
	// endpoint metadata.
	Dial(
		url *urlpkg.URL,
		prompter,
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
	url *urlpkg.URL,
	prompter,
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
	endpoint, err := handler.Dial(url, prompter, session, version, configuration, alpha)
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
		endpoint, err := connect(url, "", session, version, configuration, alpha)

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
