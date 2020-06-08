package tunnel

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
	tunnelingsvc "github.com/mutagen-io/mutagen/pkg/service/tunneling"
)

// resumeWithSelection performs a resume operation using the provided daemon
// connection and tunnel selection.
func resumeWithSelection(
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

	// Perform the resume operation, cancel prompting, and handle errors.
	tunnelingService := tunnelingsvc.NewTunnelingClient(daemonConnection)
	request := &tunnelingsvc.ResumeRequest{
		Prompter:  prompter,
		Selection: selection,
	}
	response, err := tunnelingService.Resume(context.Background(), request)
	promptingCancel()
	<-promptingErrors
	if err != nil {
		statusLinePrinter.BreakIfNonEmpty()
		return grpcutil.PeelAwayRPCErrorLayer(err)
	} else if err = response.EnsureValid(); err != nil {
		statusLinePrinter.BreakIfNonEmpty()
		return errors.Wrap(err, "invalid resume response received")
	}

	// Success.
	statusLinePrinter.Clear()
	return nil
}

// resumeMain is the entry point for the resume command.
func resumeMain(_ *cobra.Command, arguments []string) error {
	// Create tunnel selection specification.
	selection := &selection.Selection{
		All:            resumeConfiguration.all,
		Specifications: arguments,
		LabelSelector:  resumeConfiguration.labelSelector,
	}
	if err := selection.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid tunnel selection specification")
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.Connect(true, true)
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Perform the resume operation.
	return resumeWithSelection(daemonConnection, selection)
}

// resumeCommand is the resume command.
var resumeCommand = &cobra.Command{
	Use:          "resume [<tunnel>...]",
	Short:        "Resume a paused or disconnected tunnel",
	RunE:         resumeMain,
	SilenceUsage: true,
}

// resumeConfiguration stores configuration for the resume command.
var resumeConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
	// all indicates whether or not all tunnels should be resumed.
	all bool
	// labelSelector encodes a label selector to be used in identifying which
	// tunnels should be paused.
	labelSelector string
}

func init() {
	// Grab a handle for the command line flags.
	flags := resumeCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&resumeConfiguration.help, "help", "h", false, "Show help information")

	// Wire up resume flags.
	flags.BoolVarP(&resumeConfiguration.all, "all", "a", false, "Resume all tunnels")
	flags.StringVar(&resumeConfiguration.labelSelector, "label-selector", "", "Resume tunnels matching the specified label selector")
}
