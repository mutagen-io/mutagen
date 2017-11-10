package main

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/daemon"
	"github.com/havoc-io/mutagen/rpc"
	sessionpkg "github.com/havoc-io/mutagen/session"
	"github.com/havoc-io/mutagen/sync"
)

var listUsage = `usage: mutagen list [-h|--help] [-m|--monitor] [<session>]

Lists existing synchronization sessions and their statuses. A specific session
identifier can be specified to show information for only that session. If
coupled with the -m/--monitor flag, the list command will show a dynamic display
of synchronization status for the specified session.
`

func printSession(monitor bool, state sessionpkg.SessionState) {
	// Print the session identifier.
	fmt.Println("Session:", state.Session.Identifier)

	// If we're in monitor mode, that's all the information we print.
	if monitor {
		return
	}

	// Print status.
	statusString := state.State.Status.String()
	if state.Session.Paused {
		statusString = "Paused"
	}
	fmt.Println("Status:", statusString)

	// Print the last error, if any.
	if state.State.LastError != "" {
		fmt.Println("Last error:", state.State.LastError)
	}
}

func formatConnectionStatus(connected bool) string {
	if connected {
		return "Connected"
	}
	return "Disconnected"
}

func printEndpoint(monitor, alpha bool, state sessionpkg.SessionState) {
	// Print the header and URL. We combine them in monitoring mode.
	header := "Alpha:"
	url := state.Session.Alpha
	if !alpha {
		header = "Beta:"
		url = state.Session.Beta
	}
	if monitor {
		fmt.Println(header, url.Format())
	} else {
		fmt.Println(header)
		fmt.Println("\tURL:", url.Format())
	}

	// If we're in mointor mode, that's all the information we print.
	if monitor {
		return
	}

	// Print status.
	connected := state.State.AlphaConnected
	if !alpha {
		connected = state.State.BetaConnected
	}
	fmt.Println("\tStatus:", formatConnectionStatus(connected))

	// Print problems, if any.
	problems := state.State.AlphaProblems
	if !alpha {
		problems = state.State.BetaProblems
	}
	if len(problems) > 0 {
		fmt.Println("\tProblems:")
		for _, p := range problems {
			fmt.Printf("\t\t%s: %v\n", p.Path, p.Error)
		}
	}
}

func formatEntryKind(entry *sync.Entry) string {
	if entry == nil {
		return "<non-existent>"
	} else if entry.Kind == sync.EntryKind_Directory {
		return "Directory"
	} else if entry.Kind == sync.EntryKind_File {
		return "File"
	} else {
		return "<unknown>"
	}
}

func printConflicts(conflicts []sync.Conflict) {
	// Print the header.
	fmt.Println("Conflicts:")

	// Print conflicts.
	for i, c := range conflicts {
		// Print the alpha changes.
		for _, a := range c.AlphaChanges {
			fmt.Printf(
				"\t(α) %s (%s -> %s)\n",
				a.Path,
				formatEntryKind(a.Old),
				formatEntryKind(a.New),
			)
		}

		// Print the beta changes.
		for _, b := range c.BetaChanges {
			fmt.Printf(
				"\t(β) %s (%s -> %s)\n",
				b.Path,
				formatEntryKind(b.Old),
				formatEntryKind(b.New),
			)
		}

		// If we're not on the last conflict, print a newline.
		if i < len(conflicts)-1 {
			fmt.Println()
		}
	}
}

func printMonitorLine(state sessionpkg.SessionState) {
	// Print out a carriage return to wipe out the previous line.
	fmt.Print("\r")

	// Build the status line.
	var status string
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
		staging := state.State.Status == sessionpkg.SynchronizationStatusStagingAlpha ||
			state.State.Status == sessionpkg.SynchronizationStatusStagingBeta
		if staging && state.State.Staging.Total > 0 {
			status += fmt.Sprintf(
				": %.0f%% (%d/%d)",
				float32(state.State.Staging.Received)/float32(state.State.Staging.Total),
				state.State.Staging.Received,
				state.State.Staging.Total,
			)
		}
	}

	// Print the status. Ensure that it prints exactly 80 characters, truncating
	// or right-padding with space as necessary.
	fmt.Printf("%-80.80s", status)
}

func listMain(arguments []string) error {
	// Parse command line arguments.
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
	printSessionInformation := true
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

		// The first time through the loop (which will be the only time if not
		// monitoring), we print the session state.
		if printSessionInformation {
			// Loop through and print sessions.
			for i, s := range response.Sessions {
				// Print the session information.
				printSession(monitor, s)

				// Print alpha information.
				printEndpoint(monitor, true, s)

				// Print beta information.
				printEndpoint(monitor, false, s)

				// Print conflicts (if any) if we're not in monitor mode.
				if !monitor && len(s.State.Conflicts) > 0 {
					printConflicts(s.State.Conflicts)
				}

				// If we're not in monitor mode and this isn't the last session,
				// print a newline. We don't really need the monitor check since
				// there should only be one session in monitor mode, but it is
				// safer in case the daemon has sent something weird back.
				if !monitor && i < len(response.Sessions)-1 {
					fmt.Println()
				}
			}

			// Mark the session information as printed.
			printSessionInformation = false
		}

		// If we're not monitoring, we're done, otherwise print the monitoring
		// line.
		if !monitor {
			return nil
		}

		// Validate the response for monitoring. If there's an error, clear the
		// monitor line (if any) before returning for better error legibility.
		if len(response.Sessions) != 1 {
			err = errors.New("invalid listing response")
		} else if response.Sessions[0].Session.Identifier != session {
			err = errors.New("listing response returned invalid session")
		}
		if err != nil {
			if monitorLinePrinted {
				fmt.Println()
			}
			return err
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
