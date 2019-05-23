package main

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/fatih/color"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/grpcutil"
	sessionsvcpkg "github.com/havoc-io/mutagen/pkg/service/session"
	sessionpkg "github.com/havoc-io/mutagen/pkg/session"
)

func computeMonitorStatusLine(state *sessionpkg.State) string {
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
		if len(state.AlphaProblems) > 0 || len(state.BetaProblems) > 0 {
			status += color.RedString("[Problems] ")
		}

		// Add an error flag if there is one present.
		if state.LastError != "" {
			status += color.RedString("[Errored] ")
		}

		// Add the status.
		status += state.Status.Description()

		// If we're staging and have sane statistics, add them.
		if (state.Status == sessionpkg.Status_StagingAlpha ||
			state.Status == sessionpkg.Status_StagingBeta) &&
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

func monitorMain(command *cobra.Command, arguments []string) error {
	// Create session selection specification. If we don't extract an explicit
	// session specifier now, then it'll be determined automatically after the
	// first listing.
	var session string
	if len(arguments) == 1 {
		session = arguments[0]
	} else if len(arguments) > 1 {
		return errors.New("multiple session specification not allowed")
	}
	selection := &sessionpkg.Selection{
		All:            len(arguments) == 0 && monitorConfiguration.labelSelector == "",
		Specifications: arguments,
		LabelSelector:  monitorConfiguration.labelSelector,
	}
	if err := selection.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid session selection specification")
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := createDaemonClientConnection(true)
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Create a session service client.
	sessionService := sessionsvcpkg.NewSessionsClient(daemonConnection)

	// Create a status line printer and defer a break.
	statusLinePrinter := &cmd.StatusLinePrinter{}
	defer statusLinePrinter.BreakIfNonEmpty()

	// Loop and print monitoring information indefinitely.
	var previousStateIndex uint64
	sessionInformationPrinted := false
	for {
		// Create the list request. If there's no session specified, then we
		// need to grab all sessions and identify the most recently created one
		// for future queries.
		request := &sessionsvcpkg.ListRequest{
			Selection:          selection,
			PreviousStateIndex: previousStateIndex,
		}

		// Invoke list.
		response, err := sessionService.List(context.Background(), request)
		if err != nil {
			return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "list failed")
		} else if err = response.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid list response received")
		}

		// Validate the response and extract the relevant session state. If no
		// session has been explicitly specified and it's our first time through
		// the loop, then set up monitoring to use the one session identified by
		// the label selector or the most recently created session (which will
		// be the last in the returned results).
		var state *sessionpkg.State
		previousStateIndex = response.StateIndex
		if session == "" {
			if len(response.SessionStates) == 0 {
				if monitorConfiguration.labelSelector != "" {
					err = errors.New("no matching sessions exist")
				} else {
					err = errors.New("no sessions exist")
				}
			} else if monitorConfiguration.labelSelector != "" && len(response.SessionStates) > 1 {
				err = errors.New("label selector matched multiple sessions")
			} else {
				state = response.SessionStates[len(response.SessionStates)-1]
				session = state.Session.Identifier
				selection = &sessionpkg.Selection{
					Specifications: []string{session},
				}
			}
		} else if len(response.SessionStates) != 1 {
			err = errors.New("invalid list response")
		} else {
			state = response.SessionStates[0]
		}
		if err != nil {
			return err
		}

		// Print session information the first time through the loop.
		if !sessionInformationPrinted {
			// Print session information.
			printSession(state, monitorConfiguration.long)

			// Print endpoint URLs, but only if not in long mode (where they're
			// already printed in the session metadata).
			if !monitorConfiguration.long {
				fmt.Println("Alpha:", state.Session.Alpha.Format("\n\t"))
				fmt.Println("Beta:", state.Session.Beta.Format("\n\t"))
			}

			// Mark session information as printed.
			sessionInformationPrinted = true
		}

		// Compute the status line.
		statusLine := computeMonitorStatusLine(state)

		// Print the status line.
		statusLinePrinter.Print(statusLine)
	}
}

var monitorCommand = &cobra.Command{
	Use:   "monitor [<session>]",
	Short: "Shows a dynamic status display for a single session",
	Run:   cmd.Mainify(monitorMain),
}

var monitorConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
	// long indicates whether or not to use long-format monitoring.
	long bool
	// labelSelector encodes a label selector to be used in identifying which
	// sessions should be paused.
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
	flags.BoolVarP(&monitorConfiguration.long, "long", "l", false, "Show detailed session information")
	flags.StringVar(&monitorConfiguration.labelSelector, "label-selector", "", "Monitor the most recently created session matching the specified label selector")
}
