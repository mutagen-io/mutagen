package sync

import (
	"context"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"
	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	"github.com/mutagen-io/mutagen/pkg/selection"
	synchronizationsvc "github.com/mutagen-io/mutagen/pkg/service/synchronization"
)

// FlushWithLabelSelector is an orchestration convenience method that invokes
// the flush command using the specified label selector.
func FlushWithLabelSelector(labelSelector string, skipWait bool) error {
	flushConfiguration.labelSelector = labelSelector
	flushConfiguration.skipWait = skipWait
	return flushMain(nil, nil)
}

// FlushWithSessionIdentifiers is an orchestration convenience method that
// invokes the flush command with the specified session identifiers.
func FlushWithSessionIdentifiers(sessions []string, skipWait bool) error {
	flushConfiguration.skipWait = skipWait
	return flushMain(nil, sessions)
}

func flushMain(command *cobra.Command, arguments []string) error {
	// Create session selection specification.
	selection := &selection.Selection{
		All:            flushConfiguration.all,
		Specifications: arguments,
		LabelSelector:  flushConfiguration.labelSelector,
	}
	if err := selection.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid session selection specification")
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.CreateClientConnection(true, true)
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Create a session service client.
	sessionService := synchronizationsvc.NewSynchronizationClient(daemonConnection)

	// Invoke the session flush method. The stream will close when the
	// associated context is cancelled.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := sessionService.Flush(ctx)
	if err != nil {
		return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to invoke flush")
	}

	// Send the initial request.
	request := &synchronizationsvc.FlushRequest{
		Selection: selection,
		SkipWait:  flushConfiguration.skipWait,
	}
	if err := stream.Send(request); err != nil {
		return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send flush request")
	}

	// Create a status line printer.
	statusLinePrinter := &cmd.StatusLinePrinter{}

	// Receive and process responses until we're done.
	for {
		if response, err := stream.Recv(); err != nil {
			statusLinePrinter.BreakIfNonEmpty()
			return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "flush failed")
		} else if err = response.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid flush response received")
		} else if response.Message == "" {
			statusLinePrinter.Clear()
			return nil
		} else if response.Message != "" {
			statusLinePrinter.Print(response.Message)
			if err := stream.Send(&synchronizationsvc.FlushRequest{}); err != nil {
				statusLinePrinter.BreakIfNonEmpty()
				return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send message response")
			}
		}
	}
}

var flushCommand = &cobra.Command{
	Use:          "flush [<session>...]",
	Short:        "Force a synchronization cycle",
	RunE:         flushMain,
	SilenceUsage: true,
}

var flushConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
	// all indicates whether or not all sessions should be flushed.
	all bool
	// labelSelector encodes a label selector to be used in identifying which
	// sessions should be paused.
	labelSelector string
	// skipWait indicates whether or not the flush operation should block until
	// a synchronization cycle completes for each sesion requested.
	skipWait bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := flushCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&flushConfiguration.help, "help", "h", false, "Show help information")

	// Wire up flush flags.
	flags.BoolVarP(&flushConfiguration.all, "all", "a", false, "Flush all sessions")
	flags.StringVar(&flushConfiguration.labelSelector, "label-selector", "", "Flush sessions matching the specified label selector")
	flags.BoolVar(&flushConfiguration.skipWait, "skip-wait", false, "Avoid waiting for the resulting synchronization cycle(s) to complete")
}
