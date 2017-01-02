package main

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/daemon"
	"github.com/havoc-io/mutagen/rpc"
	sessionpkg "github.com/havoc-io/mutagen/session"
)

var terminateUsage = `usage: mutagen terminate [-h|--help] <session>

Terminates a synchronization session. To temporarily halt session
synchronization, use the pause command.
`

func terminateMain(arguments []string) error {
	// Parse flags.
	flagSet := cmd.NewFlagSet("terminate", terminateUsage, []int{1})
	session := flagSet.ParseOrDie(arguments)[0]
	if session == "" {
		return errors.New("empty session identifier")
	}

	// Create a daemon client.
	daemonClient := rpc.NewClient(daemon.NewOpener())

	// Invoke the session terminate method and ensure the resulting stream is
	// closed when we're done.
	stream, err := daemonClient.Invoke(sessionpkg.MethodTerminate)
	if err != nil {
		return errors.Wrap(err, "unable to invoke session terminate")
	}
	defer stream.Close()

	// Send the terminate request.
	if err := stream.Send(sessionpkg.TerminateRequest{Session: session}); err != nil {
		return errors.Wrap(err, "unable to send terminate request")
	}

	// Receive the terminate response.
	var response sessionpkg.TerminateResponse
	if err := stream.Receive(&response); err != nil {
		return errors.Wrap(err, "unable to receive terminate response")
	}

	// Success.
	return nil
}
