package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/housekeeping"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
	"github.com/mutagen-io/mutagen/pkg/synchronization/endpoint/remote"
)

const (
	// housekeepingInterval is the interval at which housekeeping will be
	// invoked by the agent.
	housekeepingInterval = 24 * time.Hour
)

func housekeepRegularly(context context.Context, logger *logging.Logger) {
	// Perform an initial housekeeping operation since the ticker won't fire
	// straight away.
	logger.Info("Performing initial housekeeping")
	housekeeping.Housekeep()

	// Create a ticker to regulate housekeeping and defer its shutdown.
	ticker := time.NewTicker(housekeepingInterval)
	defer ticker.Stop()

	// Loop and wait for the ticker or cancellation.
	for {
		select {
		case <-context.Done():
			return
		case <-ticker.C:
			logger.Info("Performing regular housekeeping")
			housekeeping.Housekeep()
		}
	}
}

func synchronizerMain(_ *cobra.Command, _ []string) error {
	// Create a channel to track termination signals. We do this before creating
	// and starting other infrastructure so that we can ensure things terminate
	// smoothly, not mid-initialization.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)

	// Set up regular housekeeping and defer its shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go housekeepRegularly(ctx, logging.RootLogger.Sublogger("housekeeping"))

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
			logging.RootLogger.Sublogger("synchronization"),
			connection,
		)
	}()

	// Wait for termination from a signal or the synchronizer.
	select {
	case sig := <-signalTermination:
		return errors.Errorf("terminated by signal: %s", sig)
	case err := <-synchronizationTermination:
		return errors.Wrap(err, "synchronization terminated")
	}
}

var synchronizerCommand = &cobra.Command{
	Use:          agent.ModeSynchronizer,
	Short:        "Run the agent in synchronizer mode",
	Args:         cmd.DisallowArguments,
	RunE:         synchronizerMain,
	SilenceUsage: true,
}

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
