package main

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/daemon"
	"github.com/havoc-io/mutagen/rpc"
	sessionpkg "github.com/havoc-io/mutagen/session"
)

var resumeUsage = `usage: mutagen resume [-h|--help] <session>

Resumes a synchronization session. This command is used to resume paused
sessions as well as provide authentication information to sessions that can't
automatically reconnect without it.
`

func resumeMain(arguments []string) error {
	// Parse command line arguments.
	flagSet := cmd.NewFlagSet("resume", resumeUsage, []int{1})
	session := flagSet.ParseOrDie(arguments)[0]
	if session == "" {
		return errors.New("empty session identifier")
	}

	// Create a daemon client.
	daemonClient := rpc.NewClient(daemon.NewOpener())

	// Invoke the session resume method and ensure the resulting stream is closed
	// when we're done.
	stream, err := daemonClient.Invoke(sessionpkg.MethodResume)
	if err != nil {
		return errors.Wrap(err, "unable to invoke session resume")
	}
	defer stream.Close()

	// Send the resume request.
	if err := stream.Send(sessionpkg.ResumeRequest{Session: session}); err != nil {
		return errors.Wrap(err, "unable to send resume request")
	}

	// Handle authentication challenges.
	if err := handlePromptRequests(stream); err != nil {
		return errors.Wrap(err, "unable to handle prompt requests")
	}

	// Receive the resume response.
	var response sessionpkg.ResumeResponse
	if err := stream.Receive(&response); err != nil {
		return errors.Wrap(err, "unable to receive resume response")
	}

	// Success.
	return nil
}
