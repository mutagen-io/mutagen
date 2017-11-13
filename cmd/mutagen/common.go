package main

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/rpc"
	"github.com/havoc-io/mutagen/session"
	"github.com/havoc-io/mutagen/ssh"
)

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
