package sync

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fatih/color"

	"google.golang.org/grpc"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/common/templating"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"

	synchronizationmodels "github.com/mutagen-io/mutagen/pkg/api/models/synchronization"
	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	"github.com/mutagen-io/mutagen/pkg/selection"
	synchronizationsvc "github.com/mutagen-io/mutagen/pkg/service/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
	"github.com/mutagen-io/mutagen/pkg/url"
)

// formatPath formats a path for display.
func formatPath(path string) string {
	if path == "" {
		return "<root>"
	}
	return path
}

// formatConnectionStatus formats a connection status for display.
func formatConnectionStatus(connected bool) string {
	if connected {
		return "Connected"
	}
	return "Disconnected"
}

// printEndpointStatus prints the status of a synchronization endpoint.
func printEndpointStatus(
	name string, url *url.URL, connected bool,
	scanProblems []*core.Problem, excludedScanProblems uint64,
	transitionProblems []*core.Problem, excludedTransitionProblems uint64,
) {
	// Print header.
	fmt.Printf("%s:\n", name)

	// Print URL if we're not in long-listing mode (otherwise it will be
	// printed elsewhere).
	if !listConfiguration.long {
		fmt.Println("\tURL:", url.Format("\n\t\t"))
	}

	// Print connection status.
	fmt.Printf("\tConnection state: %s\n", formatConnectionStatus(connected))

	// Print scan problems, if any.
	if len(scanProblems) > 0 {
		color.Red("\tScan problems:\n")
		for _, p := range scanProblems {
			color.Red("\t\t%s: %v\n", formatPath(p.Path), p.Error)
		}
		if excludedScanProblems > 0 {
			color.Red(fmt.Sprintf("\t\t...+%d more...\n", excludedScanProblems))
		}
	}

	// Print transition problems, if any.
	if len(transitionProblems) > 0 {
		color.Red("\tTransition problems:\n")
		for _, p := range transitionProblems {
			color.Red("\t\t%s: %v\n", formatPath(p.Path), p.Error)
		}
		if excludedTransitionProblems > 0 {
			color.Red(fmt.Sprintf("\t\t...+%d more...\n", excludedTransitionProblems))
		}
	}
}

// printSessionStatus prints the status of a synchronization session.
func printSessionStatus(state *synchronization.State) {
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

// formatEntry formats an entry for display.
func formatEntry(entry *core.Entry) string {
	if entry == nil {
		return "<non-existent>"
	} else if entry.Kind == core.EntryKind_Directory {
		return "Directory"
	} else if entry.Kind == core.EntryKind_File {
		if entry.Executable {
			return fmt.Sprintf("Executable File (%x)", entry.Digest)
		}
		return fmt.Sprintf("File (%x)", entry.Digest)
	} else if entry.Kind == core.EntryKind_SymbolicLink {
		return fmt.Sprintf("Symbolic Link (%s)", entry.Target)
	} else if entry.Kind == core.EntryKind_Untracked {
		return "Untracked content"
	} else if entry.Kind == core.EntryKind_Problematic {
		return fmt.Sprintf("Problematic content (%s)", entry.Problem)
	}
	return "<unknown>"
}

// printConflicts prints a list of synchronization conflicts.
func printConflicts(conflicts []*core.Conflict, excludedConflicts uint64) {
	// Print the header.
	color.Red("Conflicts:\n")

	// Print conflicts.
	for i, c := range conflicts {
		// Print the alpha changes.
		for _, a := range c.AlphaChanges {
			color.Red(
				"\t(alpha) %s (%s -> %s)\n",
				formatPath(a.Path),
				formatEntry(a.Old),
				formatEntry(a.New),
			)
		}

		// Print the beta changes.
		for _, b := range c.BetaChanges {
			color.Red(
				"\t(beta)  %s (%s -> %s)\n",
				formatPath(b.Path),
				formatEntry(b.Old),
				formatEntry(b.New),
			)
		}

		// If we're not on the last conflict, or if there are conflicts that
		// have been excluded, then print a newline.
		if i < len(conflicts)-1 || excludedConflicts > 0 {
			fmt.Println()
		}
	}

	// Print excluded conflicts.
	if excludedConflicts > 0 {
		color.Red(fmt.Sprintf("\t...+%d more...\n", excludedConflicts))
	}
}

// ListWithSelection is an orchestration convenience method that performs a list
// operation using the provided daemon connection and session selection and then
// prints status information.
func ListWithSelection(
	daemonConnection *grpc.ClientConn,
	selection *selection.Selection,
	long bool,
) error {
	// Load the formatting template (if any has been specified).
	template, err := listConfiguration.TemplateFlags.LoadTemplate()
	if err != nil {
		return fmt.Errorf("unable to load formatting template: %w", err)
	}

	// Perform the list operation.
	synchronizationService := synchronizationsvc.NewSynchronizationClient(daemonConnection)
	request := &synchronizationsvc.ListRequest{
		Selection: selection,
	}
	response, err := synchronizationService.List(context.Background(), request)
	if err != nil {
		return grpcutil.PeelAwayRPCErrorLayer(err)
	} else if err = response.EnsureValid(); err != nil {
		return fmt.Errorf("invalid list response received: %w", err)
	}

	// If a template was specified, then use that to format output with public
	// model types, otherwise use custom formatting code.
	if template != nil {
		sessions := synchronizationmodels.ExportSessions(response.SessionStates)
		if err := template.Execute(os.Stdout, sessions); err != nil {
			return fmt.Errorf("unable to execute formatting template: %w", err)
		}
	} else {
		if len(response.SessionStates) > 0 {
			for _, state := range response.SessionStates {
				fmt.Println(cmd.DelimiterLine)
				printSession(state, long)
				printEndpointStatus(
					"Alpha", state.Session.Alpha, state.AlphaConnected,
					state.AlphaScanProblems, state.ExcludedAlphaScanProblems,
					state.AlphaTransitionProblems, state.ExcludedAlphaTransitionProblems,
				)
				printEndpointStatus(
					"Beta", state.Session.Beta, state.BetaConnected,
					state.BetaScanProblems, state.ExcludedBetaScanProblems,
					state.BetaTransitionProblems, state.ExcludedBetaTransitionProblems,
				)
				printSessionStatus(state)
				if len(state.Conflicts) > 0 {
					printConflicts(state.Conflicts, state.ExcludedConflicts)
				}
			}
			fmt.Println(cmd.DelimiterLine)
		} else {
			fmt.Println(cmd.DelimiterLine)
			fmt.Println("No synchronization sessions found")
			fmt.Println(cmd.DelimiterLine)
		}
	}

	// Success.
	return nil
}

// listMain is the entry point for the list command.
func listMain(_ *cobra.Command, arguments []string) error {
	// Create session selection specification.
	selection := &selection.Selection{
		All:            len(arguments) == 0 && listConfiguration.labelSelector == "",
		Specifications: arguments,
		LabelSelector:  listConfiguration.labelSelector,
	}
	if err := selection.EnsureValid(); err != nil {
		return fmt.Errorf("invalid session selection specification: %w", err)
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.Connect(true, true)
	if err != nil {
		return fmt.Errorf("unable to connect to daemon: %w", err)
	}
	defer daemonConnection.Close()

	// Perform the list operation and print status information.
	return ListWithSelection(daemonConnection, selection, listConfiguration.long)
}

// listCommand is the list command.
var listCommand = &cobra.Command{
	Use:          "list [<session>...]",
	Short:        "List existing synchronization sessions and their statuses",
	RunE:         listMain,
	SilenceUsage: true,
}

// listConfiguration stores configuration for the list command.
var listConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
	// long indicates whether or not to use long-format listing.
	long bool
	// labelSelector encodes a label selector to be used in identifying which
	// sessions should be paused.
	labelSelector string
	// TemplateFlags store custom templating behavior.
	templating.TemplateFlags
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

	// Wire up templating flags.
	listConfiguration.TemplateFlags.Register(flags)
}
