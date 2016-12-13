package main

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/daemon"
	"github.com/havoc-io/mutagen/rpc"
	"github.com/havoc-io/mutagen/session"
	"github.com/havoc-io/mutagen/ssh"
	"github.com/havoc-io/mutagen/url"
)

var startUsage = `usage: mutagen start [-h|--help] <alpha> <beta>
`

func startMain(arguments []string) {
	// Parse and handle flags.
	flagSet := cmd.NewFlagSet("start", startUsage, []int{2})
	urls := flagSet.ParseOrDie(arguments)

	// Extract and parse URLs.
	alpha, err := url.Parse(urls[0])
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to parse alpha URL"))
	}
	beta, err := url.Parse(urls[1])
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to parse beta URL"))
	}

	// If either URL is a relative path, convert it to an absolute path.
	if alpha.Protocol == url.Protocol_Local {
		if alphaPath, err := filepath.Abs(alpha.Path); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to make alpha path absolute"))
		} else {
			alpha.Path = alphaPath
		}
	}
	if beta.Protocol == url.Protocol_Local {
		if betaPath, err := filepath.Abs(beta.Path); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to make beta path absolute"))
		} else {
			beta.Path = betaPath
		}
	}

	// Create a daemon client.
	daemonClient := rpc.NewClient(daemon.NewOpener())

	// Invoke the session start method and ensure the resulting stream is closed
	// when we're done.
	stream, err := daemonClient.Invoke(sessionMethodStart)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to invoke session creation"))
	}
	defer stream.Close()

	// Send the initial request.
	if err := stream.Encode(session.StartRequest{
		Alpha: alpha,
		Beta:  beta,
	}); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to send creation request"))
	}

	// Handle any prompts and watch for errors.
	for {
		// Grab the next response.
		var response session.StartResponse
		if err := stream.Decode(response); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to receive creation response"))
		}

		// If there is a challenge, handle it and wait for the next one.
		if response.Challenge != nil {
			if r, err := ssh.PromptCommandLine(
				response.Challenge.Message,
				response.Challenge.Prompt,
			); err != nil {
				cmd.Fatal(errors.Wrap(err, "unable to perform prompting"))
			} else if err = stream.Encode(session.StartRequest{
				Response: &session.PromptResponse{r},
			}); err != nil {
				cmd.Fatal(errors.Wrap(err, "unable to send challenge response"))
			}
			continue
		}

		// Check if there is an error.
		if response.Error != "" {
			cmd.Fatal(errors.Wrap(
				errors.New(response.Error),
				"unable to create session",
			))
		}

		// Otherwise we're done.
		break
	}
}
