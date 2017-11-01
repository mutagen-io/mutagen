package session

import (
	"context"
	"io"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/agent"
	urlpkg "github.com/havoc-io/mutagen/url"
)

type stdio struct {
	io.Reader
	io.Writer
}

func connect(
	session string,
	version Version,
	url *urlpkg.URL,
	ignores []string,
	alpha bool,
	prompter string,
) (endpoint, error) {
	// Handle based on protocol.
	if url.Protocol == urlpkg.Protocol_Local {
		// Create a local endpoint.
		endpoint, err := newLocalEndpoint(session, version, url.Path, ignores, alpha)
		if err != nil {
			return nil, errors.Wrap(err, "unable to create local endpoint")
		}

		// Success.
		return endpoint, nil
	} else if url.Protocol == urlpkg.Protocol_SSH {
		// Dial using the agent package, watching for errors
		stdInput, stdOutput, stdError, closer, err := agent.DialSSH(url, prompter, agent.ModeEndpoint)
		if err != nil {
			return nil, errors.Wrap(err, "unable to connect to SSH remote")
		}

		// Combine the processes input/output into a single stream.
		stdio := &stdio{stdOutput, stdInput}

		// Create a remote endpoint.
		endpoint, err := newRemoteEndpoint(stdio, stdError, closer, session, version, url.Path, ignores, alpha)
		if err != nil {
			return nil, errors.Wrap(err, "unable to create remote endpoint")
		}

		// Success.
		return endpoint, nil
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
func reconnect(ctx context.Context,
	session string,
	version Version,
	url *urlpkg.URL,
	ignores []string,
	alpha bool,
) (endpoint, error) {
	// Create a channel to deliver the connection result.
	results := make(chan connectResult)

	// Start a connection operation in the background.
	go func() {
		// Perform the connection.
		endpoint, err := connect(session, version, url, ignores, alpha, "")

		// If we can't transmit the resulting endpoint, close it.
		select {
		case <-ctx.Done():
			if endpoint != nil {
				endpoint.close()
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
