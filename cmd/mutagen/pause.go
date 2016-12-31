package main

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/daemon"
	"github.com/havoc-io/mutagen/rpc"
	sessionpkg "github.com/havoc-io/mutagen/session"
)

var pauseUsage = `usage: mutagen pause [-h|--help] <session>
`

func pauseMain(arguments []string) error {
	// Parse flags.
	flagSet := cmd.NewFlagSet("pause", pauseUsage, []int{1})
	session := flagSet.ParseOrDie(arguments)[0]
	if session == "" {
		return errors.New("empty session identifier")
	}

	// Create a daemon client.
	daemonClient := rpc.NewClient(daemon.NewOpener())

	// Invoke the session pause method and ensure the resulting stream is closed
	// when we're done.
	stream, err := daemonClient.Invoke(sessionpkg.MethodPause)
	if err != nil {
		return errors.Wrap(err, "unable to invoke session pause")
	}
	defer stream.Close()

	// Send the pause request.
	if err := stream.Send(sessionpkg.PauseRequest{Session: session}); err != nil {
		return errors.Wrap(err, "unable to send pause request")
	}

	// Receive the pause response.
	var response sessionpkg.PauseResponse
	if err := stream.Receive(&response); err != nil {
		return errors.Wrap(err, "unable to receive pause response")
	}

	// Success.
	return nil
}
