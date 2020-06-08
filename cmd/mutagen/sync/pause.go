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

// PauseWithSelection is an orchestration convenience method that performs a
// pause operation using the provided service client and session selection.
func PauseWithSelection(
	daemonConnection *grpc.ClientConn,
	selection *selection.Selection,
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

	// Perform the pause operation, cancel prompting, and handle errors.
	synchronizationService := synchronizationsvc.NewSynchronizationClient(daemonConnection)
	request := &synchronizationsvc.PauseRequest{
		Prompter:  prompter,
		Selection: selection,
	}
	response, err := synchronizationService.Pause(context.Background(), request)
	promptingCancel()
	<-promptingErrors
	if err != nil {
		statusLinePrinter.BreakIfNonEmpty()
		return grpcutil.PeelAwayRPCErrorLayer(err)
	} else if err = response.EnsureValid(); err != nil {
		statusLinePrinter.BreakIfNonEmpty()
		return errors.Wrap(err, "invalid pause response received")
	}

	// Success.
	statusLinePrinter.Clear()
	return nil
}

// pauseMain is the entry point for the pause command.
func pauseMain(_ *cobra.Command, arguments []string) error {
	// Create session selection specification.
	selection := &selection.Selection{
		All:            pauseConfiguration.all,
		Specifications: arguments,
		LabelSelector:  pauseConfiguration.labelSelector,
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

	// Perform the pause operation.
	return PauseWithSelection(daemonConnection, selection)
}

// pauseCommand is the pause command.
var pauseCommand = &cobra.Command{
	Use:          "pause [<session>...]",
	Short:        "Pause a synchronization session",
	RunE:         pauseMain,
	SilenceUsage: true,
}

// pauseConfiguration stores configuration for the pause command.
var pauseConfiguration struct {
	// help indicates whether or not to show help information and exit.
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
