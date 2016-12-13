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

	// Handle prompts in a separate Goroutine and watch for errors.
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
