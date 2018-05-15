package main

import (
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	sessionpkg "github.com/havoc-io/mutagen/pkg/session"
)

func resumeMain(command *cobra.Command, arguments []string) {
	// Parse session specification.
	var sessionQueries []string
	if len(arguments) > 0 {
		if resumeConfiguration.all {
			cmd.Fatal(errors.New("-a/--all specified with specific sessions"))
		}
		sessionQueries = arguments
	} else if !resumeConfiguration.all {
		cmd.Fatal(errors.New("no sessions specified"))
	}

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
	request := sessionpkg.ResumeRequest{
		All:            resumeConfiguration.all,
		SessionQueries: sessionQueries,
	}
	if err := stream.Send(request); err != nil {
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
	Use:   "resume [<session>...]",
	Short: "Resumes a paused or disconnected synchronization session",
	Run:   resumeMain,
}

var resumeConfiguration struct {
	all  bool
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := resumeCommand.Flags()
	flags.BoolVarP(&resumeConfiguration.all, "all", "a", false, "Resume all sessions")
	flags.BoolVarP(&resumeConfiguration.help, "help", "h", false, "Show help information")
}
