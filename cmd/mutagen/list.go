package main

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

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

func printSession(state *sessionpkg.State) {
	// Print the session identifier.
	fmt.Println("Session:", state.Session.Identifier)

	// Print status.
	statusString := state.Status.Description()
	if state.Session.Paused {
		statusString = "Paused"
	}
	fmt.Println("Status:", statusString)

	// Printed ignore paths, if any.
	if len(state.Session.Ignores) > 0 {
		fmt.Println("Ignored paths:")
		for _, p := range state.Session.Ignores {
			fmt.Printf("\t%s\n", p)
		}
	}

	// Print the last error, if any.
	if state.LastError != "" {
		fmt.Println("Last error:", state.LastError)
	}
}

func formatConnectionStatus(connected bool) string {
	if connected {
		return "Connected"
	}
	return "Disconnected"
}

func printEndpoint(state *sessionpkg.State, alpha bool) {
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
		fmt.Println("\tProblems:")
		for _, p := range problems {
			fmt.Printf("\t\t%s: %v\n", formatPath(p.Path), p.Error)
		}
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

func printConflicts(conflicts []*sessionpkg.Conflict) {
	// Print the header.
	fmt.Println("Conflicts:")

	// Print conflicts.
	for i, c := range conflicts {
		// Print the alpha changes.
		for _, a := range c.AlphaChanges {
			fmt.Printf(
				"\t(α) %s (%s -> %s)\n",
				formatPath(a.Path),
				formatEntryKind(a.Old),
				formatEntryKind(a.New),
			)
		}

		// Print the beta changes.
		for _, b := range c.BetaChanges {
			fmt.Printf(
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

func listMain(command *cobra.Command, arguments []string) {
	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := createDaemonClientConnection()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to connect to daemon"))
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
		cmd.Fatal(errors.Wrap(err, "unable to invoke list"))
	}

	// Determine whether or not to print delimiters.
	printDelimiters := len(response.SessionStates) > 1

	// Loop through and print sessions.
	for _, state := range response.SessionStates {
		if printDelimiters {
			fmt.Println(delimiterLine)
		}
		printSession(state)
		printEndpoint(state, true)
		printEndpoint(state, false)
		if len(state.Conflicts) > 0 {
			printConflicts(state.Conflicts)
		}
	}
	if printDelimiters {
		fmt.Println(delimiterLine)
	}
}

var listCommand = &cobra.Command{
	Use:   "list [<session>...]",
	Short: "Lists existing synchronization sessions and their statuses",
	Run:   listMain,
}

var listConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := listCommand.Flags()
	flags.BoolVarP(&listConfiguration.help, "help", "h", false, "Show help information")
}
