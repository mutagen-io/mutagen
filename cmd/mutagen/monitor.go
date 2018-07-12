package main

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/fatih/color"

	"github.com/havoc-io/mutagen/cmd"
	sessionpkg "github.com/havoc-io/mutagen/pkg/session"
	sessionsvcpkg "github.com/havoc-io/mutagen/pkg/session/service"
)

func printMonitorLine(state *sessionpkg.State) {
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

	// Print the status, prefixed with a carriage return to wipe out the
	// previous line. Ensure that the status prints as a specified width,
	// truncating or right-padding with space as necessary. On POSIX systems,
	// this width is 80 characters and on Windows it's 79. The reason for 79 on
	// Windows is that for cmd.exe consoles the line width needs to be narrower
	// than the console (which is 80 columns by default) for carriage return
	// wipes to work (if it's the same width, the next carriage return overflows
	// to the next line, behaving exactly like a newline).
	// TODO: We should probably try to detect the console width.
	fmt.Fprintf(color.Output, monitorLineFormat, status)
}

func monitorMain(command *cobra.Command, arguments []string) error {
	// Parse session specification.
	var session string
	var specifications []string
	if len(arguments) == 1 {
		session = arguments[0]
		specifications = []string{session}
	} else if len(arguments) > 1 {
		return errors.New("multiple session specification not allowed")
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := createDaemonClientConnection()
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
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
			Specifications:     specifications,
		}

		// Invoke list.
		response, err := sessionService.List(context.Background(), request)
		if err != nil {
			if monitorLinePrinted {
				fmt.Println()
			}
			return errors.Wrap(peelAwayRPCErrorLayer(err), "list failed")
		}

		// Validate the list response contents.
		for _, s := range response.SessionStates {
			if err = s.EnsureValid(); err != nil {
				if monitorLinePrinted {
					fmt.Println()
				}
				return errors.Wrap(err, "invalid session state detected in response")
			}
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
				specifications = []string{session}
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
			return err
		}

		// Print session information the first time through the loop.
		if !sessionInformationPrinted {
			// Print session information.
			printSession(state, monitorConfiguration.long)

			// Print endpoint URLs.
			fmt.Println("Alpha:", state.Session.Alpha.Format())
			fmt.Println("Beta:", state.Session.Beta.Format())

			// Mark session information as printed.
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
	Run:   cmd.Mainify(monitorMain),
}

var monitorConfiguration struct {
	help bool
	long bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := monitorCommand.Flags()
	flags.BoolVarP(&monitorConfiguration.help, "help", "h", false, "Show help information")
	flags.BoolVarP(&monitorConfiguration.long, "long", "l", false, "Show detailed session information")
}
