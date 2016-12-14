package session

import (
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
