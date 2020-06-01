package tunnel

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/fatih/color"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"
	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	selectionpkg "github.com/mutagen-io/mutagen/pkg/selection"
	tunnelingsvc "github.com/mutagen-io/mutagen/pkg/service/tunneling"
	"github.com/mutagen-io/mutagen/pkg/tunneling"
)

func computeMonitorStatusLine(state *tunneling.State) string {
	// Build the status line.
	status := "Status: "
	if state.Tunnel.Paused {
		status += color.YellowString("[Paused]")
	} else {
		// Add an error flag if there is one present.
		if state.LastError != "" {
			status += color.RedString("[Errored] ")
		}

		// Add the status.
		status += state.Status.Description()

		// If we're connected, then add session statistics.
		if state.Status == tunneling.Status_Connected {
			status += fmt.Sprintf(
				": %d active, %d total",
				state.ActiveSessions,
				state.TotalSessions,
			)
		}
	}

	// Done.
	return status
}

func monitorMain(command *cobra.Command, arguments []string) error {
	// Create a tunnel selection specification that will select our initial
	// batch of tunnels. From this batch, we'll determine which tunnel to
	// monitor based on creation date. In any case, we only allow one
	// specification to be provided in order to enforce the notion that this is
	// a single-tunnel command.
	if len(arguments) > 1 {
		return errors.New("multiple tunnel specifications not allowed")
	}
	selection := &selectionpkg.Selection{
		All:            len(arguments) == 0 && monitorConfiguration.labelSelector == "",
		Specifications: arguments,
		LabelSelector:  monitorConfiguration.labelSelector,
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

	// Create a status line printer and defer a break.
	statusLinePrinter := &cmd.StatusLinePrinter{}
	defer statusLinePrinter.BreakIfNonEmpty()

	// Loop and print monitoring information indefinitely.
	var identifier string
	var previousStateIndex uint64
	tunnelInformationPrinted := false
	for {
		// Create the list request. If there's no tunnel specified, then we
		// need to grab all tunnels and identify the most recently created one
		// for future queries.
		request := &tunnelingsvc.ListRequest{
			Selection:          selection,
			PreviousStateIndex: previousStateIndex,
		}

		// Invoke list.
		response, err := tunnelingService.List(context.Background(), request)
		if err != nil {
			return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "list failed")
		} else if err = response.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid list response received")
		}

		// Validate the response and extract the relevant tunnel state. If we
		// haven't already selected our target monitoring tunnel, then we
		// choose the last tunnel in the batch (which will be the one with the
		// most recent creation date).
		var state *tunneling.State
		previousStateIndex = response.StateIndex
		if identifier == "" {
			if len(response.TunnelStates) == 0 {
				err = errors.New("no matching tunnels exist")
			} else {
				state = response.TunnelStates[len(response.TunnelStates)-1]
				identifier = state.Tunnel.Identifier
				selection = &selectionpkg.Selection{
					Specifications: []string{identifier},
				}
			}
		} else if len(response.TunnelStates) != 1 {
			err = errors.New("invalid list response")
		} else {
			state = response.TunnelStates[0]
		}
		if err != nil {
			return err
		}

		// Print tunnel information the first time through the loop.
		if !tunnelInformationPrinted {
			// Print tunnel information.
			printTunnel(state, monitorConfiguration.long)

			// Mark tunnel information as printed.
			tunnelInformationPrinted = true
		}

		// Compute the status line.
		statusLine := computeMonitorStatusLine(state)

		// Print the status line.
		statusLinePrinter.Print(statusLine)
	}
}

var monitorCommand = &cobra.Command{
	Use:          "monitor [<tunnel>]",
	Short:        "Show a dynamic status display for a single tunnel",
	RunE:         monitorMain,
	SilenceUsage: true,
}

var monitorConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
	// long indicates whether or not to use long-format monitoring.
	long bool
	// labelSelector encodes a label selector to be used in identifying which
	// tunnels should be paused.
	labelSelector string
}

func init() {
	// Grab a handle for the command line flags.
	flags := monitorCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&monitorConfiguration.help, "help", "h", false, "Show help information")

	// Wire up monitor flags.
	flags.BoolVarP(&monitorConfiguration.long, "long", "l", false, "Show detailed tunnel information")
	flags.StringVar(&monitorConfiguration.labelSelector, "label-selector", "", "Monitor the most recently created tunnel matching the specified label selector")
}
