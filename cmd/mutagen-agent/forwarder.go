package main

import (
	"os"
	"os/signal"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/remote"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
)

func forwarderMain(command *cobra.Command, arguments []string) error {
	// Create a channel to track termination signals. We do this before creating
	// and starting other infrastructure so that we can ensure things terminate
	// smoothly, not mid-initialization.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)

	// Create a connection on standard input/output.
	connection := newStdioConnection()

	// Perform an agent handshake.
	if err := agent.ServerHandshake(connection); err != nil {
		return errors.Wrap(err, "server handshake failed")
	}

	// Perform a version handshake.
	if err := mutagen.ServerVersionHandshake(connection); err != nil {
		return errors.Wrap(err, "version handshake error")
	}

	// Serve a forwarder on standard input/output and monitor for its
	// termination.
	forwardingTermination := make(chan error, 1)
	go func() {
		forwardingTermination <- remote.ServeEndpoint(logging.RootLogger, connection)
	}()

	// Wait for termination from a signal or the forwarder.
	select {
	case sig := <-signalTermination:
		return errors.Errorf("terminated by signal: %s", sig)
	case err := <-forwardingTermination:
		return errors.Wrap(err, "forwarding terminated")
	}
}

var forwarderCommand = &cobra.Command{
	Use:          agent.ModeForwarder,
	Short:        "Run the agent in forwarder mode",
	RunE:         forwarderMain,
	SilenceUsage: true,
}

var forwarderConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := forwarderCommand.Flags()

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&forwarderConfiguration.help, "help", "h", false, "Show help information")
}
