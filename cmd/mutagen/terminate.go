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

func terminateMain(command *cobra.Command, arguments []string) error {
	// Create session selection specification.
	selection := &session.Selection{
		All:            terminateConfiguration.all,
		Specifications: arguments,
		LabelSelector:  terminateConfiguration.labelSelector,
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

	// Invoke the session terminate method. The stream will close when the
	// associated context is cancelled.
	terminateContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := sessionService.Terminate(terminateContext)
	if err != nil {
		return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to invoke terminate")
	}

	// Send the initial request.
	request := &sessionsvcpkg.TerminateRequest{
		Selection: selection,
	}
	if err := stream.Send(request); err != nil {
		return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send terminate request")
	}

	// Create a status line printer.
	statusLinePrinter := &cmd.StatusLinePrinter{}

	// Receive and process responses until we're done.
	for {
		if response, err := stream.Recv(); err != nil {
			statusLinePrinter.BreakIfNonEmpty()
			return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "terminate failed")
		} else if err = response.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid terminate response received")
		} else if response.Message == "" {
			statusLinePrinter.Clear()
			return nil
		} else if response.Message != "" {
			statusLinePrinter.Print(response.Message)
			if err := stream.Send(&sessionsvcpkg.TerminateRequest{}); err != nil {
				statusLinePrinter.BreakIfNonEmpty()
				return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send message response")
			}
		}
	}
}

var terminateCommand = &cobra.Command{
	Use:   "terminate [<session>...]",
	Short: "Permanently terminates a synchronization session",
	Run:   cmd.Mainify(terminateMain),
}

var terminateConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
	// all indicates whether or not all sessions should be terminated.
	all bool
	// labelSelector encodes a label selector to be used in identifying which
	// sessions should be paused.
	labelSelector string
}

func init() {
	// Grab a handle for the command line flags.
	flags := terminateCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&terminateConfiguration.help, "help", "h", false, "Show help information")

	// Wire up terminate flags.
	flags.BoolVarP(&terminateConfiguration.all, "all", "a", false, "Terminate all sessions")
	flags.StringVar(&terminateConfiguration.labelSelector, "label-selector", "", "Terminate sessions matching the specified label selector")
}
