package session

import (
	"context"
	"net"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/agent"
	urlpkg "github.com/havoc-io/mutagen/pkg/url"
)

func connect(
	session string,
	version Version,
	url *urlpkg.URL,
	configuration *Configuration,
	alpha bool,
	prompter string,
) (endpoint, error) {
	// Handle based on protocol.
	if url.Protocol == urlpkg.Protocol_Local {
		// Create a local endpoint.
		endpoint, err := newLocalEndpoint(session, version, url.Path, configuration, alpha)
		if err != nil {
			return nil, errors.Wrap(err, "unable to create local endpoint")
		}

		// Success.
		return endpoint, nil
	} else if url.Protocol == urlpkg.Protocol_SSH {
		if url.Hostname != "" {
			// Dial using the agent package, watching for errors
			connection, err := agent.DialSSH(url, prompter, agent.ModeEndpoint)
			if err != nil {
				return nil, errors.Wrap(err, "unable to connect to SSH remote")
			}

			// Create a remote endpoint.
			endpoint, err := newRemoteEndpoint(connection, session, version, url.Path, configuration, alpha)
			if err != nil {
				return nil, errors.Wrap(err, "unable to create remote endpoint")
			}

			// Success.
			return endpoint, nil
		} else {
			// This is a special case that we use for internal testing. An SSH URL
			// with an empty hostname is invalid, and will be rejected at any points
			// of ingress outside of Mutagen, but if it is provided internally, it
			// means to use a net.Pipe so that we can test remote endpoint
			// implementations in-memory.

			// Create a pipe.
			connection, serverConnection := net.Pipe()

			// Start an endpoint server in a separate Goroutine.
			go ServeEndpoint(serverConnection)

			// Create a new remote endpoint.
			endpoint, err := newRemoteEndpoint(connection, session, version, url.Path, configuration, alpha)
			if err != nil {
				return nil, errors.Wrap(err, "unable to create in-memory remote endpoint")
			}

			// Success.
			return endpoint, nil
		}
	} else {
		// Handle unknown protocols.
		return nil, errors.Errorf("unknown protocol: %s", url.Protocol)
	}
}

type connectResult struct {
	endpoint endpoint
	error    error
}

// reconnect is a version of connect that accepts a context for cancellation. It
// is only designed for auto-reconnection purposes, so it does not accept a
// prompter.
func reconnect(
	ctx context.Context,
	session string,
	version Version,
	url *urlpkg.URL,
	configuration *Configuration,
	alpha bool,
) (endpoint, error) {
	// Create a channel to deliver the connection result.
	results := make(chan connectResult)

	// Start a connection operation in the background.
	go func() {
		// Perform the connection.
		endpoint, err := connect(session, version, url, configuration, alpha, "")

		// If we can't transmit the resulting endpoint, shut it down.
		select {
		case <-ctx.Done():
			if endpoint != nil {
				endpoint.shutdown()
			}
		case results <- connectResult{endpoint, err}:
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
