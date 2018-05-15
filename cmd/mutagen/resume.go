package main

import (
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	sessionpkg "github.com/havoc-io/mutagen/pkg/session"
)

func resumeMain(command *cobra.Command, arguments []string) {
	// Parse session specification.
	var sessionQuery string
	if len(arguments) != 1 {
		cmd.Fatal(errors.New("session not specified"))
	}
	sessionQuery = arguments[0]

	// Create a daemon client and defer its closure.
	daemonClient, err := createDaemonClient()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create daemon client"))
	}
	defer daemonClient.Close()

	// Invoke the session resume method and ensure the resulting stream is closed
	// when we're done.
	stream, err := daemonClient.Invoke(sessionpkg.MethodResume)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to invoke session resume"))
	}
	defer stream.Close()

	// Send the resume request.
	if err := stream.Send(sessionpkg.ResumeRequest{SessionQuery: sessionQuery}); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to send resume request"))
	}

	// Handle authentication challenges.
	if err := handlePromptRequests(stream); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to handle prompt requests"))
	}

	// Receive the resume response.
	var response sessionpkg.ResumeResponse
	if err := stream.Receive(&response); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to receive resume response"))
	}
}

var resumeCommand = &cobra.Command{
	Use:   "resume <session>",
	Short: "Resumes a paused or disconnected synchronization session",
	Run:   resumeMain,
}

var resumeConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := resumeCommand.Flags()
	flags.BoolVarP(&resumeConfiguration.help, "help", "h", false, "Show help information")
}
