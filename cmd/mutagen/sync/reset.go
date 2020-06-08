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

// ResetWithSelection is an orchestration convenience method that performs a
// reset operation using the provided daemon connection and session selection.
func ResetWithSelection(
	daemonConnection *grpc.ClientConn,
	selection *selection.Selection,
) error {
	// Initiate command line prompting.
	statusLinePrinter := &cmd.StatusLinePrinter{}
	promptingCtx, promptingCancel := context.WithCancel(context.Background())
	prompter, promptingErrors, err := promptingsvc.Host(
		promptingCtx, promptingsvc.NewPromptingClient(daemonConnection),
		&cmd.StatusLinePrompter{Printer: statusLinePrinter}, true,
	)
	if err != nil {
		promptingCancel()
		return errors.Wrap(err, "unable to initiate prompting")
	}

	// Perform the reset operation, cancel prompting, and handle errors.
	synchronizationService := synchronizationsvc.NewSynchronizationClient(daemonConnection)
	request := &synchronizationsvc.ResetRequest{
		Prompter:  prompter,
		Selection: selection,
	}
	response, err := synchronizationService.Reset(context.Background(), request)
	promptingCancel()
	<-promptingErrors
	if err != nil {
		statusLinePrinter.BreakIfNonEmpty()
		return grpcutil.PeelAwayRPCErrorLayer(err)
	} else if err = response.EnsureValid(); err != nil {
		statusLinePrinter.BreakIfNonEmpty()
		return errors.Wrap(err, "invalid reset response received")
	}

	// Success.
	statusLinePrinter.Clear()
	return nil
}

// resetMain is the entry point for the reset command.
func resetMain(_ *cobra.Command, arguments []string) error {
	// Create session selection specification.
	selection := &selection.Selection{
		All:            resetConfiguration.all,
		Specifications: arguments,
		LabelSelector:  resetConfiguration.labelSelector,
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

	// Perform the reset operation.
	return ResetWithSelection(daemonConnection, selection)
}

// resetCommand is the reset command.
var resetCommand = &cobra.Command{
	Use:          "reset [<session>...]",
	Short:        "Reset synchronization session history",
	RunE:         resetMain,
	SilenceUsage: true,
}

// resetConfiguration stores configuration for the reset command.
var resetConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
	// all indicates whether or not all sessions should be reset.
	all bool
	// labelSelector encodes a label selector to be used in identifying which
	// sessions should be paused.
	labelSelector string
}

func init() {
	// Grab a handle for the command line flags.
	flags := resetCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&resetConfiguration.help, "help", "h", false, "Show help information")

	// Wire up reset flags.
	flags.BoolVarP(&resetConfiguration.all, "all", "a", false, "Reset all sessions")
	flags.StringVar(&resetConfiguration.labelSelector, "label-selector", "", "Reset sessions matching the specified label selector")
}
