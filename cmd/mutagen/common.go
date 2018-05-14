package main

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/daemon"
	"github.com/havoc-io/mutagen/pkg/rpc"
	"github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/ssh"
)

func createDaemonClient() (*rpc.Client, error) {
	// Dial the daemon.
	connection, err := daemon.DialTimeout(daemon.DefaultDialTimeout)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to daemon")
	}

	// Create the client.
	client, err := rpc.NewClient(connection)
	if err != nil {
		return nil, err
	}

	// Success.
	return client, nil
}

func handlePromptRequests(stream rpc.ClientStream) error {
	// Loop until there's an error or no more challenges.
	for {
		// Grab the next challenge.
		var challenge session.PromptRequest
		if err := stream.Receive(&challenge); err != nil {
			return errors.Wrap(err, "unable to receive authentication challenge")
		}

		// Check for completion.
		if challenge.Done {
			return nil
		}

		// Perform prompting.
		response, err := ssh.PromptCommandLine(
			challenge.Message,
			challenge.Prompt,
		)
		if err != nil {
			return errors.Wrap(err, "unable to perform prompting")
		}

		// Send the response.
		if err = stream.Send(session.PromptResponse{Response: response}); err != nil {
			return errors.Wrap(err, "unable to send challenge response")
		}
	}
}
