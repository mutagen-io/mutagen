package main

import (
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	sessionpkg "github.com/havoc-io/mutagen/pkg/session"
)

func pauseMain(command *cobra.Command, arguments []string) {
	// Parse session specification.
	var sessionQueries []string
	if len(arguments) > 0 {
		if pauseConfiguration.all {
			cmd.Fatal(errors.New("-a/--all specified with specific sessions"))
		}
		sessionQueries = arguments
	} else if !pauseConfiguration.all {
		cmd.Fatal(errors.New("no sessions specified"))
	}

	// Create a daemon client and defer its closure.
	daemonClient, err := createDaemonClient()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create daemon client"))
	}
	defer daemonClient.Close()

	// Invoke the session pause method and ensure the resulting stream is closed
	// when we're done.
	stream, err := daemonClient.Invoke(sessionpkg.MethodPause)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to invoke session pause"))
	}
	defer stream.Close()

	// Send the pause request.
	request := sessionpkg.PauseRequest{
		All:            pauseConfiguration.all,
		SessionQueries: sessionQueries,
	}
	if err := stream.Send(request); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to send pause request"))
	}

	// Receive the pause response.
	var response sessionpkg.PauseResponse
	if err := stream.Receive(&response); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to receive pause response"))
	}
}

var pauseCommand = &cobra.Command{
	Use:   "pause [<session>...]",
	Short: "Pauses a synchronization session",
	Run:   pauseMain,
}

var pauseConfiguration struct {
	all  bool
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := pauseCommand.Flags()
	flags.BoolVarP(&pauseConfiguration.all, "all", "a", false, "Pause all sessions")
	flags.BoolVarP(&pauseConfiguration.help, "help", "h", false, "Show help information")
}
