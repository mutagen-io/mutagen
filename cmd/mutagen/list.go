package main

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/daemon"
	"github.com/havoc-io/mutagen/rpc"
	"github.com/havoc-io/mutagen/session"
)

var listUsage = `usage: mutagen list [-h|--help] [<session>]
`

func printSessionState(state session.SessionState, long bool) {
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

func listMain(arguments []string) error {
	// Parse flags.
	flagSet := cmd.NewFlagSet("list", listUsage, []int{0, 1})
	sessionArguments := flagSet.ParseOrDie(arguments)
	var filteredSession string
	if len(sessionArguments) == 1 {
		filteredSession = sessionArguments[0]
	}

	// Create a daemon client.
	daemonClient := rpc.NewClient(daemon.NewOpener())

	// Invoke the session list method and ensure the resulting stream is closed
	// when we're done.
	stream, err := daemonClient.Invoke(session.MethodList)
	if err != nil {
		return errors.Wrap(err, "unable to invoke session enumeration")
	}
	defer stream.Close()

	// Send the list request and receive the response.
	var response session.ListResponse
	if err := stream.Encode(session.ListRequest{}); err != nil {
		return errors.Wrap(err, "unable to send enumeration request")
	} else if err = stream.Decode(&response); err != nil {
		return errors.Wrap(err, "unable to receive session list")
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
		return errors.New("unable to find specified session")
	}

	// Success.
	return nil
}
