package main

import (
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/daemon"
	"github.com/havoc-io/mutagen/pkg/rpc"
	sessionpkg "github.com/havoc-io/mutagen/pkg/session"
)

func terminateMain(command *cobra.Command, arguments []string) {
	// Parse session specification.
	var session string
	if len(arguments) != 1 {
		cmd.Fatal(errors.New("session not specified"))
	}
	session = arguments[0]

	// Create a daemon client.
	daemonClient := rpc.NewClient(daemon.NewOpener())

	// Invoke the session terminate method and ensure the resulting stream is
	// closed when we're done.
	stream, err := daemonClient.Invoke(sessionpkg.MethodTerminate)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to invoke session terminate"))
	}
	defer stream.Close()

	// Send the terminate request.
	if err := stream.Send(sessionpkg.TerminateRequest{Session: session}); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to send terminate request"))
	}

	// Receive the terminate response.
	var response sessionpkg.TerminateResponse
	if err := stream.Receive(&response); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to receive terminate response"))
	}
}

var terminateCommand = &cobra.Command{
	Use:   "terminate <session>",
	Short: "Permanently terminates a synchronization session",
	Run:   terminateMain,
}

var terminateConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := terminateCommand.Flags()
	flags.BoolVarP(&terminateConfiguration.help, "help", "h", false, "Show help information")
}
