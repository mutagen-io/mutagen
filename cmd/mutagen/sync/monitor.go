package sync

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fatih/color"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/common/templating"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"

	synchronizationmodels "github.com/mutagen-io/mutagen/pkg/api/models/synchronization"
	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	selectionpkg "github.com/mutagen-io/mutagen/pkg/selection"
	synchronizationsvc "github.com/mutagen-io/mutagen/pkg/service/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
)

// computeMonitorStatusLine constructs a monitoring status line for a
// synchronization session.
func computeMonitorStatusLine(state *synchronization.State) string {
	// Build the status line.
	status := "Status: "
	if state.Session.Paused {
		status += color.YellowString("[Paused]")
	} else {
		// Add a conflict flag if there are conflicts.
		if len(state.Conflicts) > 0 {
			status += color.RedString("[Conflicts] ")
		}

		// Add a problems flag if there are problems.
		haveProblems := len(state.AlphaScanProblems) > 0 ||
			len(state.BetaScanProblems) > 0 ||
			len(state.AlphaTransitionProblems) > 0 ||
			len(state.BetaTransitionProblems) > 0
		if haveProblems {
			status += color.RedString("[Problems] ")
		}

		// Add an error flag if there is one present.
		if state.LastError != "" {
			status += color.RedString("[Errored] ")
		}

		// Add the status.
		status += state.Status.Description()

		// If we're staging and have sane statistics, add them.
		if (state.Status == synchronization.Status_StagingAlpha ||
			state.Status == synchronization.Status_StagingBeta) &&
			state.StagingStatus != nil {
			status += fmt.Sprintf(
				": %.0f%% (%d/%d)",
				100.0*float32(state.StagingStatus.Received)/float32(state.StagingStatus.Total),
				state.StagingStatus.Received,
				state.StagingStatus.Total,
			)
		}
	}

	// Done.
	return status
}

// monitorMain is the entry point for the monitor command.
func monitorMain(_ *cobra.Command, arguments []string) error {
	// Create the session selection specification that will select our initial
	// batch of sessions.
	selection := &selectionpkg.Selection{
		All:            len(arguments) == 0 && monitorConfiguration.labelSelector == "",
		Specifications: arguments,
		LabelSelector:  monitorConfiguration.labelSelector,
	}
	if err := selection.EnsureValid(); err != nil {
		return fmt.Errorf("invalid session selection specification: %w", err)
	}

	// Load the formatting template (if any has been specified).
	template, err := monitorConfiguration.TemplateFlags.LoadTemplate()
	if err != nil {
		return fmt.Errorf("unable to load formatting template: %w", err)
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.Connect(true, true)
	if err != nil {
		return fmt.Errorf("unable to connect to daemon: %w", err)
	}
	defer daemonConnection.Close()

	// Create a session service client.
	sessionService := synchronizationsvc.NewSynchronizationClient(daemonConnection)

	// Create the list request that we'll use.
	request := &synchronizationsvc.ListRequest{
		Selection: selection,
	}

	// If no template has been specified, then create a status line printer and
	// defer a line break operation.
	var statusLinePrinter *cmd.StatusLinePrinter
	if template == nil {
		statusLinePrinter = &cmd.StatusLinePrinter{}
		defer statusLinePrinter.BreakIfNonEmpty()
	}

	// Track whether or not we've identified an individual session in the
	// non-templated case.
	var identifiedSingleTargetSession bool

	// Loop and print monitoring information indefinitely.
	for {
		// Perform a list operation.
		response, err := sessionService.List(context.Background(), request)
		if err != nil {
			return fmt.Errorf("list failed: %w", grpcutil.PeelAwayRPCErrorLayer(err))
		} else if err = response.EnsureValid(); err != nil {
			return fmt.Errorf("invalid list response received: %w", err)
		}

		// Update the state tracking index.
		request.PreviousStateIndex = response.StateIndex

		// If a template has been specified, then use that to format output with
		// public model types. No validation is necessary here since we don't
		// require any specific number of sessions.
		if template != nil {
			sessions := synchronizationmodels.ExportSessions(response.SessionStates)
			if err := template.Execute(os.Stdout, sessions); err != nil {
				return fmt.Errorf("unable to execute formatting template: %w", err)
			}
			continue
		}

		// No template has been specified, but our command line monitoring
		// interface only supports dynamic status displays for a single session
		// at a time, so we choose the newest session identified by the initial
		// criteria and update our selection to target it specifically.
		var state *synchronization.State
		if !identifiedSingleTargetSession {
			if len(response.SessionStates) == 0 {
				err = errors.New("no matching sessions exist")
			} else {
				// Select the most recently created session matching the
				// selection criteria (which are ordered by creation date).
				state = response.SessionStates[len(response.SessionStates)-1]

				// Update the selection criteria to target only that session.
				request.Selection = &selectionpkg.Selection{
					Specifications: []string{state.Session.Identifier},
				}

				// Print session information.
				printSession(state, monitorConfiguration.long)

				// Print endpoint URLs, but only if not in long mode (where
				// they're already printed in the session metadata).
				if !monitorConfiguration.long {
					fmt.Println("Alpha:", state.Session.Alpha.Format("\n\t"))
					fmt.Println("Beta:", state.Session.Beta.Format("\n\t"))
				}

				// Record that we've identified our target session.
				identifiedSingleTargetSession = true
			}
		} else if len(response.SessionStates) != 1 {
			err = errors.New("invalid list response")
		} else {
			state = response.SessionStates[0]
		}
		if err != nil {
			return err
		}

		// Compute the status line.
		statusLine := computeMonitorStatusLine(state)

		// Print the status line.
		statusLinePrinter.Print(statusLine)
	}
}

// monitorCommand is the monitor command.
var monitorCommand = &cobra.Command{
	Use:          "monitor [<session>...]",
	Short:        "Display streaming session status information",
	RunE:         monitorMain,
	SilenceUsage: true,
}

// monitorConfiguration stores configuration for the monitor command.
var monitorConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
	// long indicates whether or not to use long-format monitoring.
	long bool
	// labelSelector encodes a label selector to be used in identifying which
	// sessions should be paused.
	labelSelector string
	// TemplateFlags store custom templating behavior.
	templating.TemplateFlags
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
	flags.BoolVarP(&monitorConfiguration.long, "long", "l", false, "Show detailed session information")
	flags.StringVar(&monitorConfiguration.labelSelector, "label-selector", "", "Monitor the most recently created session matching the specified label selector")

	// Wire up templating flags.
	monitorConfiguration.TemplateFlags.Register(flags)
}
