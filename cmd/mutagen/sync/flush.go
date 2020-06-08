package sync

import (
	"context"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"google.golang.org/grpc"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"

	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	"github.com/mutagen-io/mutagen/pkg/selection"
	promptingsvc "github.com/mutagen-io/mutagen/pkg/service/prompting"
	synchronizationsvc "github.com/mutagen-io/mutagen/pkg/service/synchronization"
)

// FlushWithSelection is an orchestration convenience method that performs a
// flush operation using the provided daemon connection and session selection.
func FlushWithSelection(
	daemonConnection *grpc.ClientConn,
	selection *selection.Selection,
	skipWait bool,
) error {
	// Initiate command line messaging.
	statusLinePrinter := &cmd.StatusLinePrinter{}
	promptingCtx, promptingCancel := context.WithCancel(context.Background())
	prompter, promptingErrors, err := promptingsvc.Host(
		promptingCtx, promptingsvc.NewPromptingClient(daemonConnection),
		&cmd.StatusLinePrompter{Printer: statusLinePrinter}, false,
	)
	if err != nil {
		promptingCancel()
		return errors.Wrap(err, "unable to initiate prompting")
	}

	// Perform the flush operation, cancel prompting, and handle errors.
	synchronizationService := synchronizationsvc.NewSynchronizationClient(daemonConnection)
	request := &synchronizationsvc.FlushRequest{
		Prompter:  prompter,
		Selection: selection,
		SkipWait:  skipWait,
	}
	response, err := synchronizationService.Flush(context.Background(), request)
	promptingCancel()
	<-promptingErrors
	if err != nil {
		statusLinePrinter.BreakIfNonEmpty()
		return grpcutil.PeelAwayRPCErrorLayer(err)
	} else if err = response.EnsureValid(); err != nil {
		statusLinePrinter.BreakIfNonEmpty()
		return errors.Wrap(err, "invalid flush response received")
	}

	// Success.
	statusLinePrinter.Clear()
	return nil
}

// flushMain is the entry point for the flush command.
func flushMain(_ *cobra.Command, arguments []string) error {
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
	daemonConnection, err := daemon.Connect(true, true)
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Perform the flush operation.
	return FlushWithSelection(daemonConnection, selection, flushConfiguration.skipWait)
}

// flushCommand is the flush command.
var flushCommand = &cobra.Command{
	Use:          "flush [<session>...]",
	Short:        "Force a synchronization cycle",
	RunE:         flushMain,
	SilenceUsage: true,
}

// flushConfiguration stores configuration for the flush command.
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
