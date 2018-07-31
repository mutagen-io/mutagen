package main

import (
	"context"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	sessionsvcpkg "github.com/havoc-io/mutagen/pkg/session/service"
)

func pauseMain(command *cobra.Command, arguments []string) error {
	// Parse session specifications.
	var specifications []string
	if len(arguments) > 0 {
		if pauseConfiguration.all {
			return errors.New("-a/--all specified with specific sessions")
		}
		specifications = arguments
	} else if !pauseConfiguration.all {
		return errors.New("no sessions specified")
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := createDaemonClientConnection()
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Create a session service client.
	sessionService := sessionsvcpkg.NewSessionsClient(daemonConnection)

	// Invoke the session pause method. The stream will close when the
	// associated context is cancelled.
	pauseContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := sessionService.Pause(pauseContext)
	if err != nil {
		return errors.Wrap(peelAwayRPCErrorLayer(err), "unable to invoke pause")
	}

	// Send the initial request.
	request := &sessionsvcpkg.PauseRequest{
		Specifications: specifications,
	}
	if err := stream.Send(request); err != nil {
		return errors.Wrap(peelAwayRPCErrorLayer(err), "unable to send pause request")
	}

	// Create a status line printer.
	statusLinePrinter := &cmd.StatusLinePrinter{}

	// Receive and process responses until we're done.
	for {
		if response, err := stream.Recv(); err != nil {
			statusLinePrinter.BreakIfNonEmpty()
			return errors.Wrap(peelAwayRPCErrorLayer(err), "pause failed")
		} else if response.Message == "" {
			statusLinePrinter.Clear()
			return nil
		} else if response.Message != "" {
			statusLinePrinter.Print(response.Message)
			if err := stream.Send(&sessionsvcpkg.PauseRequest{}); err != nil {
				statusLinePrinter.BreakIfNonEmpty()
				return errors.Wrap(peelAwayRPCErrorLayer(err), "unable to send message response")
			}
		}
	}

	// Success.
	return nil
}

var pauseCommand = &cobra.Command{
	Use:   "pause [<session>...]",
	Short: "Pauses a synchronization session",
	Run:   cmd.Mainify(pauseMain),
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
