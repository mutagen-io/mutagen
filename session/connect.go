package session

import (
	"context"
	"io"
	"net"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/agent"
	"github.com/havoc-io/mutagen/url"
)

func connect(remote *url.URL, prompter string) (io.ReadWriteCloser, error) {
	// Handle based on protocol.
	if remote.Protocol == url.Protocol_Local {
		// Create an in-memory pipe.
		clientConnection, serverConnection := net.Pipe()

		// Start the endpoint on the server end.
		go ServeEndpoint(serverConnection)

		// Success.
		return clientConnection, nil
	} else if remote.Protocol == url.Protocol_SSH {
		// Dial using the agent package, watching for errors
		connection, err := agent.DialSSH(remote, prompter, agent.ModeEndpoint)
		if err != nil {
			return nil, errors.Wrap(err, "unable to connect to SSH remote")
		}

		// Success.
		return connection, nil
	} else {
		// Handle unknown protocols.
		return nil, errors.Errorf("unknown protocol: %s", remote.Protocol)
	}
}

type connectResult struct {
	connection io.ReadWriteCloser
	error      error
}

// reconnect is a version of connect that accepts a context for cancellation. It
// is only designed for auto-reconnection purposes, so it does not accept a
// prompter.
func reconnect(ctx context.Context, remote *url.URL) (io.ReadWriteCloser, error) {
	// Create a channel to deliver the connection result.
	results := make(chan connectResult)

	// Start a connection operation in the background.
	go func() {
		// Perform the connection.
		connection, err := connect(remote, "")

		// If we can't transmit the connection, close it.
		select {
		case <-ctx.Done():
			if connection != nil {
				connection.Close()
			}
		case results <- connectResult{connection, err}:
		}
	}()

	// Wait for context cancellation or results.
	select {
	case <-ctx.Done():
		return nil, errors.New("reconnect cancelled")
	case result := <-results:
		return result.connection, result.error
	}
}
