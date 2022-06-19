package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"

	"github.com/fatih/color"

	"github.com/dustin/go-humanize"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/common"
	"github.com/mutagen-io/mutagen/cmd/mutagen/common/templating"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"

	synchronizationmodels "github.com/mutagen-io/mutagen/pkg/api/models/synchronization"
	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	selectionpkg "github.com/mutagen-io/mutagen/pkg/selection"
	synchronizationsvc "github.com/mutagen-io/mutagen/pkg/service/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/rsync"
)

// computeMonitorStatusLine constructs a monitoring status line for a
// synchronization session.
func computeMonitorStatusLine(state *synchronization.State) string {
	// Build the status line.
	var status string
	if state.Session.Paused {
		status += color.YellowString("[Paused]")
	} else {
		// Add a conflict flag if there are conflicts.
		if len(state.Conflicts) > 0 {
			status += color.YellowString("[C] ")
		}

		// Add a problems flag if there are problems.
		haveProblems := len(state.AlphaState.ScanProblems) > 0 ||
			len(state.BetaState.ScanProblems) > 0 ||
			len(state.AlphaState.TransitionProblems) > 0 ||
			len(state.BetaState.TransitionProblems) > 0
		if haveProblems {
			status += color.YellowString("[!] ")
		}

		// Add an error flag if there is one present.
		if state.LastError != "" {
			status += color.RedString("[X] ")
		}

		// Handle the formatting based on status. If we're in a staging mode,
		// then extract the relevant progress information. Despite not having a
		// built-in mechanism for knowing the total expected size of a staging
		// operation, we do know the number of files that the staging operation
		// is performing, so if that's equal to the number of files on the
		// source endpoint, then we know that we can use the total file size on
		// the source endpoint as an estimate for the total staging size.
		var stagingProgress *rsync.ReceiverState
		var totalExpectedSize uint64
		if state.Status == synchronization.Status_StagingAlpha {
			status += "[←] "
			stagingProgress = state.AlphaState.StagingProgress
			if stagingProgress == nil {
				status += "Preparing to stage files on alpha"
			} else if stagingProgress.ExpectedFiles == state.BetaState.FileCount {
				totalExpectedSize = state.BetaState.TotalFileSize
			}
		} else if state.Status == synchronization.Status_StagingBeta {
			status += "[→] "
			stagingProgress = state.BetaState.StagingProgress
			if stagingProgress == nil {
				status += "Preparing to stage files on beta"
			} else if stagingProgress.ExpectedFiles == state.AlphaState.FileCount {
				totalExpectedSize = state.AlphaState.TotalFileSize
			}
		} else {
			status += state.Status.Description()
		}

		// Print staging progress, if available.
		if stagingProgress != nil {
			var fractionComplete float32
			var totalSizeDenominator string
			if totalExpectedSize != 0 {
				fractionComplete = float32(stagingProgress.TotalReceivedSize) / float32(totalExpectedSize)
				totalSizeDenominator = "/" + humanize.Bytes(totalExpectedSize)
			} else {
				fractionComplete = float32(stagingProgress.ReceivedFiles) / float32(stagingProgress.ExpectedFiles)
			}
			status += fmt.Sprintf("[%d/%d - %s%s - %.0f%%] %s (%s/%s)",
				stagingProgress.ReceivedFiles, stagingProgress.ExpectedFiles,
				humanize.Bytes(stagingProgress.TotalReceivedSize), totalSizeDenominator,
				100.0*fractionComplete,
				path.Base(stagingProgress.Path),
				humanize.Bytes(stagingProgress.ReceivedSize), humanize.Bytes(stagingProgress.ExpectedSize),
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

	// Determine the listing mode.
	mode := common.SessionDisplayModeMonitor
	if monitorConfiguration.long {
		mode = common.SessionDisplayModeMonitorLong
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

	// Track the last update time.
	var lastUpdateTime time.Time

	// Track whether or not we've identified an individual session in the
	// non-templated case.
	var identifiedSingleTargetSession bool

	// Loop and print monitoring information indefinitely.
	for {
		// Regulate the update frequency (and tame CPU usage in both the monitor
		// command and the daemon) by enforcing a minimum update cycle interval.
		now := time.Now()
		timeSinceLastUpdate := now.Sub(lastUpdateTime)
		if timeSinceLastUpdate < common.MinimumMonitorUpdateInterval {
			time.Sleep(common.MinimumMonitorUpdateInterval - timeSinceLastUpdate)
		}
		lastUpdateTime = now

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
				printSession(state, mode)

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
