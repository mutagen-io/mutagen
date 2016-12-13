package main

import (
	"fmt"

	"github.com/pkg/errors"

	"golang.org/x/net/context"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/session"
)

var listUsage = `usage: mutagen list [-h|--help] [<session>]
`

func printSessionState(state *session.SessionState, long bool) {
	// Print the session identifier.
	fmt.Println(state.Session.Identifier)

	// Print endpoints.
	// TODO: Add connectivity status.
	fmt.Println("Alpha:", state.Session.Alpha)
	fmt.Println("Beta:", state.Session.Beta)

	// TODO: Implement.
	fmt.Println(state)

	// Print an empty line.
	fmt.Println()
}

func listMain(arguments []string) {
	// Parse flags.
	flagSet := cmd.NewFlagSet("list", listUsage, []int{0, 1})
	sessionArguments := flagSet.ParseOrDie(arguments)
	var filteredSession string
	if len(sessionArguments) == 1 {
		filteredSession = sessionArguments[0]
	}

	// Create a daemon client connection and defer its closure.
	daemonClientConnection, err := newDaemonClientConnection()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to connect to daemon"))
	}
	defer daemonClientConnection.Close()

	// Create a session manager client.
	sessionsClient := session.NewSessionsClient(daemonClientConnection)

	// Perform a list request.
	response, err := sessionsClient.List(context.Background(), &session.ListRequest{})
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to list sessions"))
	}

	// Print sessions. If there's a filter, only print that session, but print
	// it long-form. If nothing matches the filter, treat it as an error.
	matchFound := false
	for _, s := range response.Sessions {
		if filteredSession != "" {
			if s.Session.Identifier != filteredSession {
				continue
			}
			printSessionState(s, true)
			matchFound = true
			break
		} else {
			printSessionState(s, false)
		}
	}
	if filteredSession != "" && !matchFound {
		cmd.Fatal(errors.New("unable to find specified session"))
	}
}
