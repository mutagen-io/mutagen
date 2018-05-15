package main

import (
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	sessionpkg "github.com/havoc-io/mutagen/pkg/session"
)

func terminateMain(command *cobra.Command, arguments []string) {
	// Parse session specification.
	var sessionQueries []string
	if len(arguments) > 0 {
		if terminateConfiguration.all {
			cmd.Fatal(errors.New("-a/--all specified with specific sessions"))
		}
		sessionQueries = arguments
	} else if !terminateConfiguration.all {
		cmd.Fatal(errors.New("no sessions specified"))
	}

	// Create a daemon client and defer its closure.
	daemonClient, err := createDaemonClient()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create daemon client"))
	}
	defer daemonClient.Close()

	// Invoke the session terminate method and ensure the resulting stream is
	// closed when we're done.
	stream, err := daemonClient.Invoke(sessionpkg.MethodTerminate)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to invoke session terminate"))
	}
	defer stream.Close()

	// Send the terminate request.
	request := sessionpkg.TerminateRequest{
		All:            terminateConfiguration.all,
		SessionQueries: sessionQueries,
	}
	if err := stream.Send(request); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to send terminate request"))
	}

	// Receive the terminate response.
	var response sessionpkg.TerminateResponse
	if err := stream.Receive(&response); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to receive terminate response"))
	}
}

var terminateCommand = &cobra.Command{
	Use:   "terminate [<session>...]",
	Short: "Permanently terminates a synchronization session",
	Run:   terminateMain,
}

var terminateConfiguration struct {
	all  bool
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := terminateCommand.Flags()
	flags.BoolVarP(&terminateConfiguration.all, "all", "a", false, "Terminate all sessions")
	flags.BoolVarP(&terminateConfiguration.help, "help", "h", false, "Show help information")
}
