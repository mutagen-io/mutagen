package main

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/daemon"
	"github.com/havoc-io/mutagen/rpc"
	sessionpkg "github.com/havoc-io/mutagen/session"
	"github.com/havoc-io/mutagen/sync"
)

var listUsage = `usage: mutagen list [-h|--help] [-m|--monitor] [<session>]
`

type connectionState struct {
	alphaConnected bool
	betaConnected  bool
}

var connectionStatePrefixes = map[connectionState]string{
	{false, false}: "XX",
	{true, false}:  "-X",
	{false, true}:  "X-",
	{true, true}:   "--",
}

func monitorPrefix(state sessionpkg.SessionState) string {
	switch state.State.Status {
	case sessionpkg.SynchronizationStatusDisconnected:
		fallthrough
	case sessionpkg.SynchronizationStatusConnecting:
		return connectionStatePrefixes[connectionState{
			state.State.AlphaConnected,
			state.State.BetaConnected,
		}]
	case sessionpkg.SynchronizationStatusInitializing:
		return "**"
	case sessionpkg.SynchronizationStatusScanning:
		return "--"
	case sessionpkg.SynchronizationStatusReconciling:
		return "~~"
	case sessionpkg.SynchronizationStatusStaging:
		return "><"
	case sessionpkg.SynchronizationStatusTransitioning:
		return "<>"
	case sessionpkg.SynchronizationStatusSaving:
		return "[]"
	default:
		return "  "
	}
}

func monitorConflictSummary(conflicts []sync.Conflict) string {
	if len(conflicts) > 0 {
		return "X"
	}
	return "-"
}

func monitorProblemSummary(problems []sync.Problem) string {
	if len(problems) > 0 {
		return "X"
	}
	return "-"
}

const monitorStatusBarInnerWidth = 31

func monitorStatusBar(status sessionpkg.StagingStatus) string {
	// If there is no staging going on, then return empty spaces.
	if status.Total == 0 {
		return fmt.Sprintf("[%s]", strings.Repeat(" ", monitorStatusBarInnerWidth))
	}

	// Watch for invalid or easy status cases.
	if status.Index >= status.Total {
		return fmt.Sprintf("[%s]", strings.Repeat("#", monitorStatusBarInnerWidth))
	}

	// Compute the number of spaces meant to be occupied by completed blocks.
	fractionCompleted := float32(status.Index) / float32(status.Total)
	completedSpaces := int(fractionCompleted * monitorStatusBarInnerWidth)

	// Compute the resultant bar.
	return fmt.Sprintf(
		"[%s%s]",
		strings.Repeat("#", completedSpaces),
		strings.Repeat("-", monitorStatusBarInnerWidth-completedSpaces),
	)
}

func printMonitorLine(state sessionpkg.SessionState) {
	// Print out a carriage return to wipe out the previous line.
	fmt.Print("\r")

	// Print the state prefix and a trailing space.
	fmt.Printf("%s ", monitorPrefix(state))

	// Print the conflict status and a trailing space.
	fmt.Printf("%s ", monitorConflictSummary(state.State.Conflicts))

	// Print the alpha status bar and a trailing space.
	fmt.Printf(
		"α(%s)%s ",
		monitorProblemSummary(state.State.AlphaProblems),
		monitorStatusBar(state.State.AlphaStaging),
	)

	// Print the beta status bar.
	fmt.Printf(
		"β(%s)%s",
		monitorProblemSummary(state.State.BetaProblems),
		monitorStatusBar(state.State.BetaStaging),
	)
}

func connectionFormat(connected bool) string {
	if connected {
		return "connected"
	}
	return "disconnected"
}

func printSessionState(state sessionpkg.SessionState) {
	// Print the session identifier.
	fmt.Println(state.Session.Identifier)

	// Print status.
	fmt.Printf("Status: %s", state.State.Status)
	if state.Session.Paused {
		fmt.Print(" (Paused)")
	}
	fmt.Println()

	// Print last error if present.
	if state.State.LastError != "" {
		fmt.Println("Last synchronization error: %s", state.State.LastError)
	}

	// Print alpha information.
	// TODO: Add staging status.
	fmt.Printf("Alpha: %s (%s)\n", state.Session.Alpha.Format(), connectionFormat(state.State.AlphaConnected))

	// Print beta information.
	// TODO: Add staging status.
	fmt.Printf("Beta: %s (%s)\n", state.Session.Beta.Format(), connectionFormat(state.State.BetaConnected))

	// TODO: Print conflicts.

	// TODO: Print problems.
}

func listMain(arguments []string) error {
	// Parse flags.
	var session string
	var monitor bool
	flagSet := cmd.NewFlagSet("list", listUsage, []int{0, 1})
	flagSet.BoolVarP(&monitor, "monitor", "m", false, "continuously monitor session")
	sessionArguments := flagSet.ParseOrDie(arguments)
	if len(sessionArguments) == 1 {
		session = sessionArguments[0]
	}

	// Check that options are sane.
	if monitor && session == "" {
		return errors.New("-m/--monitor only supported with single session")
	}

	// Create a daemon client.
	daemonClient := rpc.NewClient(daemon.NewOpener())

	// Invoke the session list method and ensure the resulting stream is closed
	// when we're done.
	stream, err := daemonClient.Invoke(sessionpkg.MethodList)
	if err != nil {
		return errors.Wrap(err, "unable to invoke session listing")
	}
	defer stream.Close()

	// Send the list request.
	if err := stream.Send(sessionpkg.ListRequest{
		Session: session,
		Monitor: monitor,
	}); err != nil {
		return errors.Wrap(err, "unable to send listing request")
	}

	// Loop indefinitely. We'll bail after a single response if monitoring
	// wasn't requested.
	for {
		// Receive the next response.
		var response sessionpkg.ListResponse
		if err := stream.Receive(&response); err != nil {
			return errors.Wrap(err, "unable to receive listing response")
		}

		// Validate and print accordingly.
		if monitor {
			if len(response.Sessions) != 1 {
				return errors.New("invalid listing response")
			} else if state := response.Sessions[0]; state.Session.Identifier != session {
				return errors.New("listing response returned invalid session")
			} else {
				printMonitorLine(response.Sessions[0])
			}
		} else {
			// Loop through and print sessions.
			for _, state := range response.Sessions {
				printSessionState(state)
			}

			// Done.
			return nil
		}
	}
}
