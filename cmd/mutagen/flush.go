package main

import (
	"context"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	sessionsvcpkg "github.com/havoc-io/mutagen/pkg/service/session"
)

func flushMain(command *cobra.Command, arguments []string) error {
	// Parse session specifications.
	var specifications []string
	if len(arguments) > 0 {
		if flushConfiguration.all {
			return errors.New("-a/--all specified with specific sessions")
		}
		specifications = arguments
	} else if !flushConfiguration.all {
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

	// Invoke the session flush method. The stream will close when the
	// associated context is cancelled.
	flushContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := sessionService.Flush(flushContext)
	if err != nil {
		return errors.Wrap(peelAwayRPCErrorLayer(err), "unable to invoke flush")
	}

	// Send the initial request.
	request := &sessionsvcpkg.FlushRequest{
		Specifications: specifications,
		SkipWait:       flushConfiguration.skipWait,
	}
	if err := stream.Send(request); err != nil {
		return errors.Wrap(peelAwayRPCErrorLayer(err), "unable to send flush request")
	}

	// Create a status line printer.
	statusLinePrinter := &cmd.StatusLinePrinter{}

	// Receive and process responses until we're done.
	for {
		if response, err := stream.Recv(); err != nil {
			statusLinePrinter.BreakIfNonEmpty()
			return errors.Wrap(peelAwayRPCErrorLayer(err), "flush failed")
		} else if err = response.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid flush response received")
		} else if response.Message == "" {
			statusLinePrinter.Clear()
			return nil
		} else if response.Message != "" {
			statusLinePrinter.Print(response.Message)
			if err := stream.Send(&sessionsvcpkg.FlushRequest{}); err != nil {
				statusLinePrinter.BreakIfNonEmpty()
				return errors.Wrap(peelAwayRPCErrorLayer(err), "unable to send message response")
			}
		}
	}
}

var flushCommand = &cobra.Command{
	Use:   "flush [<session>...]",
	Short: "Flushes a synchronization session",
	Run:   cmd.Mainify(flushMain),
}

var flushConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
	// all indicates whether or not all sessions should be flushed.
	all bool
	// skipWait indicates whether or not the flush operation should block until
	// a synchronization cycle completes for each sesion requested.
	skipWait bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := flushCommand.Flags()

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&flushConfiguration.help, "help", "h", false, "Show help information")

	// Wire up flush flags.
	flags.BoolVarP(&flushConfiguration.all, "all", "a", false, "Flush all sessions")
	flags.BoolVar(&flushConfiguration.skipWait, "skip-wait", false, "Avoid waiting for the resulting synchronization cycle to complete")
}
