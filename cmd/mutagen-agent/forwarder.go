package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"

	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/remote"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
)

// forwarderMain is the entry point for the forwarder command.
func forwarderMain(_ *cobra.Command, _ []string) error {
	// Create a channel to track termination signals. We do this before creating
	// and starting other infrastructure so that we can ensure things terminate
	// smoothly, not mid-initialization.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)

	// Create a stream using standard input/output.
	stream := newStdioStream()

	// Perform an agent handshake.
	if err := agent.ServerHandshake(stream); err != nil {
		return fmt.Errorf("server handshake failed: %w", err)
	}

	// Perform a version handshake.
	if err := mutagen.ServerVersionHandshake(stream); err != nil {
		return fmt.Errorf("version handshake error: %w", err)
	}

	// Serve a forwarder on standard input/output and monitor for its
	// termination.
	forwardingTermination := make(chan error, 1)
	go func() {
		forwardingTermination <- remote.ServeEndpoint(
			logging.RootLogger.Sublogger("forwarding"),
			stream,
		)
	}()

	// Wait for termination from a signal or the forwarder.
	select {
	case sig := <-signalTermination:
		return fmt.Errorf("terminated by signal: %s", sig)
	case err := <-forwardingTermination:
		return fmt.Errorf("forwarding terminated: %w", err)
	}
}

// forwarderCommand is the forwarder command.
var forwarderCommand = &cobra.Command{
	Use:          agent.ModeForwarder,
	Short:        "Run the agent in forwarder mode",
	Args:         cmd.DisallowArguments,
	RunE:         forwarderMain,
	SilenceUsage: true,
}

// forwarderConfiguration stores configuration for the forwarder command.
var forwarderConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := forwarderCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&forwarderConfiguration.help, "help", "h", false, "Show help information")
}
