package main

import (
	"context"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/grpcutil"
	sessionsvcpkg "github.com/havoc-io/mutagen/pkg/service/session"
	"github.com/havoc-io/mutagen/pkg/session"
)

func pauseMain(command *cobra.Command, arguments []string) error {
	// Create session selection specification.
	selection := &session.Selection{
		All:            pauseConfiguration.all,
		Specifications: arguments,
		LabelSelector:  pauseConfiguration.labelSelector,
	}
	if err := selection.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid session selection specification")
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := createDaemonClientConnection(true)
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
		return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to invoke pause")
	}

	// Send the initial request.
	request := &sessionsvcpkg.PauseRequest{
		Selection: selection,
	}
	if err := stream.Send(request); err != nil {
		return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send pause request")
	}

	// Create a status line printer.
	statusLinePrinter := &cmd.StatusLinePrinter{}

	// Receive and process responses until we're done.
	for {
		if response, err := stream.Recv(); err != nil {
			statusLinePrinter.BreakIfNonEmpty()
			return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "pause failed")
		} else if err = response.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid pause response received")
		} else if response.Message == "" {
			statusLinePrinter.Clear()
			return nil
		} else if response.Message != "" {
			statusLinePrinter.Print(response.Message)
			if err := stream.Send(&sessionsvcpkg.PauseRequest{}); err != nil {
				statusLinePrinter.BreakIfNonEmpty()
				return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send message response")
			}
		}
	}
}

var pauseCommand = &cobra.Command{
	Use:   "pause [<session>...]",
	Short: "Pauses a synchronization session",
	Run:   cmd.Mainify(pauseMain),
}

var pauseConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
	// all indicates whether or not all sessions should be paused.
	all bool
	// labelSelector encodes a label selector to be used in identifying which
	// sessions should be paused.
	labelSelector string
}

func init() {
	// Grab a handle for the command line flags.
	flags := pauseCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&pauseConfiguration.help, "help", "h", false, "Show help information")

	// Wire up pause flags.
	flags.BoolVarP(&pauseConfiguration.all, "all", "a", false, "Pause all sessions")
	flags.StringVar(&pauseConfiguration.labelSelector, "label-selector", "", "Pause sessions matching the specified label selector")
}
