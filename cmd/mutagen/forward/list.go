package forward

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/fatih/color"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	"github.com/mutagen-io/mutagen/pkg/selection"
	forwardingsvc "github.com/mutagen-io/mutagen/pkg/service/forwarding"
	"github.com/mutagen-io/mutagen/pkg/url"
)

func formatConnectionStatus(connected bool) string {
	if connected {
		return "Connected"
	}
	return "Disconnected"
}

func printEndpointStatus(name string, url *url.URL, connected bool) {
	// Print header.
	fmt.Printf("%s:\n", name)

	// Print URL if we're not in long-listing mode (otherwise it will be
	// printed elsewhere).
	if !listConfiguration.long {
		fmt.Println("\tURL:", url.Format("\n\t\t"))
	}

	// Print connection status.
	fmt.Printf("\tConnection state: %s\n", formatConnectionStatus(connected))
}

func printSessionStatus(state *forwarding.State) {
	// Print status.
	statusString := state.Status.Description()
	if state.Session.Paused {
		statusString = color.YellowString("[Paused]")
	}
	fmt.Fprintln(color.Output, "Status:", statusString)

	// Print the last error, if any.
	if state.LastError != "" {
		color.Red("Last error: %s\n", state.LastError)
	}
}

// ListWithLabelSelector is an orchestration convenience method that invokes the
// list command using the specified label selector.
func ListWithLabelSelector(labelSelector string, long bool) error {
	listConfiguration.long = long
	listConfiguration.labelSelector = labelSelector
	return listMain(nil, nil)
}

func listMain(command *cobra.Command, arguments []string) error {
	// Create session selection specification.
	selection := &selection.Selection{
		All:            len(arguments) == 0 && listConfiguration.labelSelector == "",
		Specifications: arguments,
		LabelSelector:  listConfiguration.labelSelector,
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

	// Create a session service client.
	sessionService := forwardingsvc.NewForwardingClient(daemonConnection)

	// Invoke list.
	request := &forwardingsvc.ListRequest{
		Selection: selection,
	}
	response, err := sessionService.List(context.Background(), request)
	if err != nil {
		return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "list failed")
	} else if err = response.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid list response received")
	}

	// Handle output based on whether or not any sessions were returned.
	if len(response.SessionStates) > 0 {
		for _, state := range response.SessionStates {
			fmt.Println(cmd.DelimiterLine)
			printSession(state, listConfiguration.long)
			printEndpointStatus("Source", state.Session.Source, state.SourceConnected)
			printEndpointStatus("Destination", state.Session.Destination, state.DestinationConnected)
			printSessionStatus(state)
		}
		fmt.Println(cmd.DelimiterLine)
	} else {
		fmt.Println(cmd.DelimiterLine)
		fmt.Println("No sessions found")
		fmt.Println(cmd.DelimiterLine)
	}

	// Success.
	return nil
}

var listCommand = &cobra.Command{
	Use:          "list [<session>...]",
	Short:        "List existing forwarding sessions and their statuses",
	RunE:         listMain,
	SilenceUsage: true,
}

var listConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
	// long indicates whether or not to use long-format listing.
	long bool
	// labelSelector encodes a label selector to be used in identifying which
	// sessions should be paused.
	labelSelector string
}

func init() {
	// Grab a handle for the command line flags.
	flags := listCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&listConfiguration.help, "help", "h", false, "Show help information")

	// Wire up list flags.
	flags.BoolVarP(&listConfiguration.long, "long", "l", false, "Show detailed session information")
	flags.StringVar(&listConfiguration.labelSelector, "label-selector", "", "List sessions matching the specified label selector")
}
