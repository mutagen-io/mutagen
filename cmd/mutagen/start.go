package main

import (
	"path/filepath"

	"github.com/pkg/errors"

	"golang.org/x/net/context"

	"github.com/havoc-io/mutagen/cmd"
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

	// Extract URLs.
	alpha := urls[0]
	beta := urls[1]

	// If either URL is a relative path, convert it to an absolute path.
	if url.Classify(alpha) == url.TypePath {
		if alphaAbs, err := filepath.Abs(alpha); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to make first path absolute"))
		} else {
			alpha = alphaAbs
		}
	}
	if url.Classify(beta) == url.TypePath {
		if betaAbs, err := filepath.Abs(beta); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to make second path absolute"))
		} else {
			beta = betaAbs
		}
	}

	// Create a daemon client connection and defer its closure.
	daemonClientConnection, err := newDaemonClientConnection()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to connect to daemon"))
	}
	defer daemonClientConnection.Close()

	// Create a prompt service client.
	promptClient := ssh.NewPromptClient(daemonClientConnection)

	// Start responding to prompts.
	prompts, err := promptClient.Respond(context.Background(), grpcCallFlags...)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to register as prompter"))
	}

	// Receive prompter identifier.
	var prompter string
	if response, err := prompts.Recv(); err != nil || response.Prompter == "" {
		cmd.Fatal(errors.Wrap(err, "unable to receive prompter identifier"))
	} else {
		prompter = response.Prompter
	}

	// Handle prompts and watch for errors.
	promptErrors := make(chan error, 1)
	go func() {
		promptErrors <- performPrompts(prompts)
	}()

	// Create a session manager client.
	sessionsClient := session.NewSessionsClient(daemonClientConnection)

	// Invoke start and watch for completion.
	startErrors := make(chan error, 1)
	go func() {
		// Create the request.
		startRequest := &session.StartRequest{
			Alpha:    alpha,
			Beta:     beta,
			Prompter: prompter,
		}

		// Invoke start.
		_, err := sessionsClient.Start(context.Background(), startRequest, grpcCallFlags...)
		startErrors <- err
	}()

	// Wait for the start method to return or prompting to fail.
	select {
	case promptErr := <-promptErrors:
		cmd.Fatal(errors.Wrap(promptErr, "prompting failed"))
	case startErr := <-startErrors:
		if startErr != nil {
			cmd.Fatal(errors.Wrap(startErr, "unable to start session"))
		}
	}
}
