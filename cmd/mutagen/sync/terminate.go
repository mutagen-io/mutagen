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

// TerminateWithSelection is an orchestration convenience method that performs a
// terminate operation using the provided daemon connection and session
// selection.
func TerminateWithSelection(
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

	// Perform the terminate operation, cancel prompting, and handle errors.
	synchronizationService := synchronizationsvc.NewSynchronizationClient(daemonConnection)
	request := &synchronizationsvc.TerminateRequest{
		Prompter:  prompter,
		Selection: selection,
	}
	response, err := synchronizationService.Terminate(context.Background(), request)
	promptingCancel()
	<-promptingErrors
	if err != nil {
		statusLinePrinter.BreakIfNonEmpty()
		return grpcutil.PeelAwayRPCErrorLayer(err)
	} else if err = response.EnsureValid(); err != nil {
		statusLinePrinter.BreakIfNonEmpty()
		return errors.Wrap(err, "invalid terminate response received")
	}

	// Success.
	statusLinePrinter.Clear()
	return nil
}

// terminateMain is the entry point for the terminate command.
func terminateMain(_ *cobra.Command, arguments []string) error {
	// Create session selection specification.
	selection := &selection.Selection{
		All:            terminateConfiguration.all,
		Specifications: arguments,
		LabelSelector:  terminateConfiguration.labelSelector,
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

	// Perform the terminate operation.
	return TerminateWithSelection(daemonConnection, selection)
}

// terminateCommand is the terminate command.
var terminateCommand = &cobra.Command{
	Use:          "terminate [<session>...]",
	Short:        "Permanently terminate a synchronization session",
	RunE:         terminateMain,
	SilenceUsage: true,
}

// terminateConfiguration stores configuration for the terminate command.
var terminateConfiguration struct {
	// help indicates whether or not to show help information and exit.
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
