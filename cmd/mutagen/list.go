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
	"github.com/havoc-io/mutagen/pkg/sync"
)

func formatPath(path string) string {
	if path == "" {
		return "(root)"
	}
	return path
}

func formatConnectionStatus(connected bool) string {
	if connected {
		return "Connected"
	}
	return "Disconnected"
}

func printEndpointStatus(state *sessionpkg.State, alpha bool) {
	// Print the header for this endpoint.
	header := "Alpha:"
	if !alpha {
		header = "Beta:"
	}
	fmt.Println(header)

	// Print URL.
	url := state.Session.Alpha
	if !alpha {
		url = state.Session.Beta
	}
	fmt.Println("\tURL:", url.Format())

	// Print status.
	connected := state.AlphaConnected
	if !alpha {
		connected = state.BetaConnected
	}
	fmt.Println("\tStatus:", formatConnectionStatus(connected))

	// Print problems, if any.
	problems := state.AlphaProblems
	if !alpha {
		problems = state.BetaProblems
	}
	if len(problems) > 0 {
		color.Red("\tProblems:\n")
		for _, p := range problems {
			color.Red("\t\t%s: %v\n", formatPath(p.Path), p.Error)
		}
	}
}

func printSessionStatus(state *sessionpkg.State) {
	// Print status.
	statusString := state.Status.Description()
	if state.Session.Paused {
		statusString = color.YellowString("[Paused]")
	}
	fmt.Println("Status:", statusString)

	// Print the last error, if any.
	if state.LastError != "" {
		color.Red("Last error: %s\n", state.LastError)
	}
}

func formatEntryKind(entry *sync.Entry) string {
	if entry == nil {
		return "<non-existent>"
	} else if entry.Kind == sync.EntryKind_Directory {
		return "Directory"
	} else if entry.Kind == sync.EntryKind_File {
		if entry.Executable {
			return fmt.Sprintf("Executable File (%x)", entry.Digest)
		}
		return fmt.Sprintf("File (%x)", entry.Digest)
	} else if entry.Kind == sync.EntryKind_Symlink {
		return fmt.Sprintf("Symlink (%s)", entry.Target)
	} else {
		return "<unknown>"
	}
}

func printConflicts(conflicts []*sync.Conflict) {
	// Print the header.
	color.Red("Conflicts:\n")

	// Print conflicts.
	for i, c := range conflicts {
		// Print the alpha changes.
		for _, a := range c.AlphaChanges {
			color.Red(
				"\t(α) %s (%s -> %s)\n",
				formatPath(a.Path),
				formatEntryKind(a.Old),
				formatEntryKind(a.New),
			)
		}

		// Print the beta changes.
		for _, b := range c.BetaChanges {
			color.Red(
				"\t(β) %s (%s -> %s)\n",
				formatPath(b.Path),
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

func listMain(command *cobra.Command, arguments []string) error {
	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := createDaemonClientConnection()
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Create a session service client.
	sessionService := sessionsvcpkg.NewSessionClient(daemonConnection)

	// Invoke list.
	request := &sessionsvcpkg.ListRequest{
		Specifications: arguments,
	}
	response, err := sessionService.List(context.Background(), request)
	if err != nil {
		return errors.Wrap(peelAwayRPCErrorLayer(err), "list failed")
	}

	// Validate the list response contents.
	for _, s := range response.SessionStates {
		if err = s.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid session state detected in response")
		}
	}

	// Loop through and print sessions.
	for _, state := range response.SessionStates {
		fmt.Println(delimiterLine)
		printSession(state, listConfiguration.long)
		printEndpointStatus(state, true)
		printEndpointStatus(state, false)
		printSessionStatus(state)
		if len(state.Conflicts) > 0 {
			printConflicts(state.Conflicts)
		}
	}

	// Print a final delimiter line if there were any sessions.
	if len(response.SessionStates) > 0 {
		fmt.Println(delimiterLine)
	}

	// Success.
	return nil
}

var listCommand = &cobra.Command{
	Use:   "list [<session>...]",
	Short: "Lists existing synchronization sessions and their statuses",
	Run:   cmd.Mainify(listMain),
}

var listConfiguration struct {
	help bool
	long bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := listCommand.Flags()
	flags.BoolVarP(&listConfiguration.help, "help", "h", false, "Show help information")
	flags.BoolVarP(&listConfiguration.long, "long", "l", false, "Show detailed session information")
}
