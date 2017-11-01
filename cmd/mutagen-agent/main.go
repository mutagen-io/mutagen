package main

import (
	"io"
	"os"
	"os/signal"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/agent"
	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/session"
)

var agentUsage = `usage: mutagen-agent should not be manually invoked
`

type stdio struct {
	io.Reader
	io.Writer
}

func main() {
	// Parse command line arguments.
	flagSet := cmd.NewFlagSet("mutagen-agent", agentUsage, []int{1})
	mode := flagSet.ParseOrDie(os.Args[1:])[0]

	// Handle install.
	if mode == agent.ModeInstall {
		if err := agent.Install(); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to install"))
		}
		return
	}

	// Perform housekeeping.
	agent.Housekeep()
	session.HousekeepCaches()
	session.HousekeepStaging()

	// Create a stream on standard input/output.
	stdio := &stdio{os.Stdin, os.Stdout}

	// Perform a handshake.
	if err := mutagen.SendVersion(stdio); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to transmit version"))
	}

	// Handle based on mode.
	if mode == agent.ModeEndpoint {
		// Serve an endpoint on standard input/output and monitor for its
		// termination.
		endpointTermination := make(chan error, 1)
		go func() {
			endpointTermination <- session.ServeEndpoint(stdio, os.Stderr)
		}()

		// Wait for termination from a signal or the endpoint. At this point we
		// can't (or at least shouldn't) write to standard error because the
		// endpoint server is using it to stream watch events and writing to it
		// could cause the decoder on the other end to do something weird (like
		// treat our output as a message size and try to decode a bunch of data,
		// though even then it will unblock when the process exits). So we just
		// bail once termination occurs. Anywhere else in this function it's
		// fine to write to standard error because the endpoint client won't
		// have started polling standard error yet.
		signalTermination := make(chan os.Signal, 1)
		signal.Notify(signalTermination, cmd.TerminationSignals...)
		select {
		case <-signalTermination:
			cmd.Die()
		case <-endpointTermination:
			cmd.Die()
		}
	} else {
		cmd.Fatal(errors.Errorf("unknown mode: %s", mode))
	}
}
