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

func pauseMain(command *cobra.Command, arguments []string) error {
	// Create tunnel selection specification.
	selection := &selection.Selection{
		All:            pauseConfiguration.all,
		Specifications: arguments,
		LabelSelector:  pauseConfiguration.labelSelector,
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

	// Invoke the tunnel pause method. The stream will close when the
	// associated context is cancelled.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := tunnelingService.Pause(ctx)
	if err != nil {
		return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to invoke pause")
	}

	// Send the initial request.
	request := &tunnelingsvc.PauseRequest{
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
			if err := stream.Send(&tunnelingsvc.PauseRequest{}); err != nil {
				statusLinePrinter.BreakIfNonEmpty()
				return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send message response")
			}
		}
	}
}

var pauseCommand = &cobra.Command{
	Use:          "pause [<tunnel>...]",
	Short:        "Pause a tunnel",
	RunE:         pauseMain,
	SilenceUsage: true,
}

var pauseConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
	// all indicates whether or not all tunnels should be paused.
	all bool
	// labelSelector encodes a label selector to be used in identifying which
	// tunnels should be paused.
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
	flags.BoolVarP(&pauseConfiguration.all, "all", "a", false, "Pause all tunnels")
	flags.StringVar(&pauseConfiguration.labelSelector, "label-selector", "", "Pause tunnels matching the specified label selector")
}
