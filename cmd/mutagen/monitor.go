package main

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	sessionpkg "github.com/havoc-io/mutagen/pkg/session"
	sessionsvcpkg "github.com/havoc-io/mutagen/pkg/session/service"
)

func printMonitorLine(state *sessionpkg.State) {
	// Build the status line.
	status := "Status: "
	if state.Session.Paused {
		status += "Paused"
	} else {
		// Add a conflict flag if there are conflicts.
		if len(state.Conflicts) > 0 {
			status += "[Conflicts] "
		}

		// Add a problems flag if there are problems.
		if len(state.AlphaProblems) > 0 || len(state.BetaProblems) > 0 {
			status += "[Problems] "
		}

		// Add the status.
		status += state.Status.Description()

		// If we're staging and have sane statistics, add them.
		if state.Status == sessionpkg.Status_StagingAlpha ||
			state.Status == sessionpkg.Status_StagingBeta &&
				state.StagingStatus.Total > 0 {
			status += fmt.Sprintf(
				": %.0f%% (%d/%d)",
				100.0*float32(state.StagingStatus.Received)/float32(state.StagingStatus.Total),
				state.StagingStatus.Received,
				state.StagingStatus.Total,
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

func monitorMain(command *cobra.Command, arguments []string) {
	// Parse session specification.
	var session string
	var sessionQueries []string
	if len(arguments) == 1 {
		session = arguments[0]
		sessionQueries = []string{session}
	} else if len(arguments) > 1 {
		cmd.Fatal(errors.New("multiple session specification not allowed"))
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := createDaemonClientConnection()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to connect to daemon"))
	}
	defer daemonConnection.Close()

	// Create a session service client.
	sessionService := sessionsvcpkg.NewSessionClient(daemonConnection)

	// Loop and print monitoring information indefinitely.
	var previousStateIndex uint64
	sessionInformationPrinted := false
	monitorLinePrinted := false
	for {
		// Create the list request. If there's no session specified, then we
		// need to grab all sessions and identify the most recently created one
		// for future queries.
		request := &sessionsvcpkg.ListRequest{
			PreviousStateIndex: previousStateIndex,
			All:                session == "",
			SessionQueries:     sessionQueries,
		}

		// Invoke list.
		response, err := sessionService.List(context.Background(), request)
		if err != nil {
			if monitorLinePrinted {
				fmt.Println()
			}
			cmd.Fatal(errors.Wrap(err, "unable to invoke list"))
		}

		// Validate the response and extract the relevant session state. If no
		// session has been specified and it's our first time through the loop,
		// identify the most recently created session.
		var state *sessionpkg.State
		previousStateIndex = response.StateIndex
		if session == "" {
			if len(response.SessionStates) == 0 {
				err = errors.New("no sessions exist")
			} else {
				state = response.SessionStates[len(response.SessionStates)-1]
				session = state.Session.Identifier
				sessionQueries = []string{session}
			}
		} else if len(response.SessionStates) != 1 {
			err = errors.New("invalid list response")
		} else {
			state = response.SessionStates[0]
		}
		if err != nil {
			if monitorLinePrinted {
				fmt.Println()
			}
			cmd.Fatal(err)
		}

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
		printMonitorLine(state)
		monitorLinePrinted = true
	}
}

var monitorCommand = &cobra.Command{
	Use:   "monitor [<session>]",
	Short: "Shows a dynamic status display for the specified session",
	Run:   monitorMain,
}

var monitorConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := monitorCommand.Flags()
	flags.BoolVarP(&monitorConfiguration.help, "help", "h", false, "Show help information")
}
