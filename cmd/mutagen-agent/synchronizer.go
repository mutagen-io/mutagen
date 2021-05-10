package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"

	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/housekeeping"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
	"github.com/mutagen-io/mutagen/pkg/synchronization/endpoint/remote"
)

// synchronizerMain is the entry point for the synchronizer command.
func synchronizerMain(_ *cobra.Command, _ []string) error {
	// Create a channel to track termination signals. We do this before creating
	// and starting other infrastructure so that we can ensure things terminate
	// smoothly, not mid-initialization.
	terminationSignals := make(chan os.Signal, 1)
	signal.Notify(terminationSignals, cmd.TerminationSignals...)

	// Create the root logger.
	logger := logging.NewLogger(os.Stderr)

	// Set up regular housekeeping and defer its shutdown.
	housekeepingCtx, cancelHousekeeping := context.WithCancel(context.Background())
	defer cancelHousekeeping()
	go housekeeping.HousekeepRegularly(housekeepingCtx, logger.Sublogger("housekeeping"))

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

	// Serve a synchronizer on standard input/output and monitor for its
	// termination.
	synchronizationTermination := make(chan error, 1)
	go func() {
		synchronizationTermination <- remote.ServeEndpoint(
			logger.Sublogger("synchronization"),
			connection,
		)
	}()

	// Wait for termination from a signal or the synchronizer.
	select {
	case s := <-terminationSignals:
		return errors.Errorf("terminated by signal: %s", s)
	case err := <-synchronizationTermination:
		return errors.Wrap(err, "synchronization terminated")
	}
}

// synchronizerCommand is the synchronizer command.
var synchronizerCommand = &cobra.Command{
	Use:          agent.ModeSynchronizer,
	Short:        "Run the agent in synchronizer mode",
	Args:         cmd.DisallowArguments,
	RunE:         synchronizerMain,
	SilenceUsage: true,
}

// synchronizerConfiguration stores configuration for the synchronizer command.
var synchronizerConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := synchronizerCommand.Flags()

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&synchronizerConfiguration.help, "help", "h", false, "Show help information")
}
