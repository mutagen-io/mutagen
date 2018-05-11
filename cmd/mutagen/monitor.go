package main

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/daemon"
	"github.com/havoc-io/mutagen/pkg/rpc"
	sessionpkg "github.com/havoc-io/mutagen/pkg/session"
)

var monitorUsage = `usage: mutagen monitor [-h|--help] [<session>]

Shows a dynamic status display for the specified session. If no session is
specified, then the most recently created session is displayed.
`

func printMonitorLine(state sessionpkg.SessionState) {
	// Build the status line.
	status := "Status: "
	if state.Session.Paused {
		status += "Paused"
	} else {
		// Add a conflict flag if there are conflicts.
		if len(state.State.Conflicts) > 0 {
			status += "[Conflicts] "
		}

		// Add a problems flag if there are problems.
		if len(state.State.AlphaProblems) > 0 || len(state.State.BetaProblems) > 0 {
			status += "[Problems] "
		}

		// Add the status.
		status += state.State.Status.String()

		// If we're staging and have sane statistics, add them.
		if state.State.Status == sessionpkg.SynchronizationStatusStagingAlpha ||
			state.State.Status == sessionpkg.SynchronizationStatusStagingBeta &&
				state.State.Staging.Total > 0 {
			status += fmt.Sprintf(
				": %.0f%% (%d/%d)",
				100.0*float32(state.State.Staging.Received)/float32(state.State.Staging.Total),
				state.State.Staging.Received,
				state.State.Staging.Total,
			)
		}
	}

	// Print the status, prefixed with a carriage return to wipe out the
	// previous line. Ensure that the status prints as a specified width,
	// truncating or right-padding with space as necessary. On POSIX systems,
	// this width is 80 characters and on Windows it's 79. The reason for 79 on
	// Windows is that for cmd.exe consoles the line width needs to be narrower
	// than the console (which is 80 columns by default) for carriage return
	// wipes to work (if it's the same width, the next carriage return overflows
	// to the next line, behaving exactly like a newline).
	// TODO: We should probably try to detect the console width.
	fmt.Printf(monitorLineFormat, status)
}

func monitorMain(arguments []string) error {
	// Parse command line arguments.
	var session string
	flagSet := cmd.NewFlagSet("monitor", monitorUsage, []int{0, 1})
	sessions := flagSet.ParseOrDie(arguments)
	if len(sessions) == 1 {
		session = sessions[0]
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
	kind := sessionpkg.ListRequestKindRepeated
	if session == "" {
		kind = sessionpkg.ListRequestKindRepeatedLatest
	}
	request := sessionpkg.ListRequest{Kind: kind, Session: session}
	if err := stream.Send(request); err != nil {
		return errors.Wrap(err, "unable to send listing request")
	}

	// Loop indefinitely. We'll bail after a single response if monitoring
	// wasn't requested.
	sessionInformationPrinted := false
	monitorLinePrinted := false
	for {
		// Receive the next response. If there's an error, clear the monitor
		// line (if any) before returning for better error legibility.
		var response sessionpkg.ListResponse
		if err := stream.Receive(&response); err != nil {
			if monitorLinePrinted {
				fmt.Println()
			}
			return errors.Wrap(err, "unable to receive listing response")
		}

		// Validate the response. If there's an error, clear the monitor line
		// (if any) before returning for better error legibility.
		if len(response.Sessions) != 1 {
			err = errors.New("invalid listing response")
		}
		if err != nil {
			if monitorLinePrinted {
				fmt.Println()
			}
			return err
		}

		// Extract the session state.
		state := response.Sessions[0]

		// Print session information the first time through the loop.
		if !sessionInformationPrinted {
			fmt.Println("Session:", state.Session.Identifier)
			if len(state.Session.Ignores) > 0 {
				fmt.Println("Ignored paths:")
				for _, p := range state.Session.Ignores {
					fmt.Printf("\t%s\n", p)
				}
			}
			fmt.Println("Alpha:", state.Session.Alpha.Format())
			fmt.Println("Beta:", state.Session.Beta.Format())
			sessionInformationPrinted = true
		}

		// Print the monitoring line and record that we've done so.
		printMonitorLine(response.Sessions[0])
		monitorLinePrinted = true

		// Send another (empty) request to let the daemon know that we're ready
		// for another response. This is a backpressure mechanism to keep the
		// daemon from sending more requests than we can handle in monitor mode.
		// If there's an error, clear the monitor line (if any) before returning
		// for better error legibility.
		if err := stream.Send(sessionpkg.ListRequest{}); err != nil {
			if monitorLinePrinted {
				fmt.Println()
			}
			return errors.Wrap(err, "unable to send ready request")
		}
	}
}
