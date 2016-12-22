package main

import (
	"os"
	"os/signal"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/agent"
	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/session"
	"github.com/havoc-io/mutagen/stream"
)

var agentUsage = `usage: mutagen-agent should not be manually invoked
`

func main() {
	// Parse flags.
	flagSet := cmd.NewFlagSet("mutagen-agent", agentUsage, []int{1})
	mode := flagSet.ParseOrDie(os.Args[1:])[0]

	// Handle install.
	if mode == agent.ModeInstall {
		if err := agent.Install(); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to install"))
		}
		return
	}

	// Create a stream on standard input/output.
	stdio := stream.New(os.Stdin, os.Stdout, os.Stdout)

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
			endpointTermination <- session.ServeEndpoint(stdio)
		}()

		// Wait for termination from a signal or the endpoint.
		signalTermination := make(chan os.Signal, 1)
		signal.Notify(signalTermination, cmd.TerminationSignals...)
		select {
		case sig := <-signalTermination:
			cmd.Fatal(errors.Errorf("terminated by signal: %s", sig))
		case err := <-endpointTermination:
			cmd.Fatal(errors.Wrap(err, "endpoint terminated"))
		}
	} else {
		cmd.Fatal(errors.Errorf("unknown mode: %s", mode))
	}
}
