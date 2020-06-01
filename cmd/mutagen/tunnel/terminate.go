package tunnel

import (
	"context"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"
	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	"github.com/mutagen-io/mutagen/pkg/selection"
	tunnelingsvc "github.com/mutagen-io/mutagen/pkg/service/tunneling"
)

func terminateMain(command *cobra.Command, arguments []string) error {
	// Create tunnel selection specification.
	selection := &selection.Selection{
		All:            terminateConfiguration.all,
		Specifications: arguments,
		LabelSelector:  terminateConfiguration.labelSelector,
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

	// Create a tunneling service client.
	tunnelingService := tunnelingsvc.NewTunnelingClient(daemonConnection)

	// Invoke the tunnel terminate method. The stream will close when the
	// associated context is cancelled.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := tunnelingService.Terminate(ctx)
	if err != nil {
		return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to invoke terminate")
	}

	// Send the initial request.
	request := &tunnelingsvc.TerminateRequest{
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
			if err := stream.Send(&tunnelingsvc.TerminateRequest{}); err != nil {
				statusLinePrinter.BreakIfNonEmpty()
				return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send message response")
			}
		}
	}
}

var terminateCommand = &cobra.Command{
	Use:          "terminate [<tunnel>...]",
	Short:        "Permanently terminate a tunnel",
	RunE:         terminateMain,
	SilenceUsage: true,
}

var terminateConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
	// all indicates whether or not all tunnels should be terminated.
	all bool
	// labelSelector encodes a label selector to be used in identifying which
	// tunnels should be paused.
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
	flags.BoolVarP(&terminateConfiguration.all, "all", "a", false, "Terminate all tunnels")
	flags.StringVar(&terminateConfiguration.labelSelector, "label-selector", "", "Terminate tunnels matching the specified label selector")
}
