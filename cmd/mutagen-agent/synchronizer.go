package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

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

// housekeepRegularly is the entry point for the housekeeping Goroutine.
func housekeepRegularly(ctx context.Context, logger *logging.Logger) {
	// Perform an initial housekeeping operation since the ticker won't fire
	// straight away.
	logger.Info("Performing initial housekeeping")
	housekeeping.Housekeep(logger)

	// Create a ticker to regulate housekeeping and defer its shutdown.
	ticker := time.NewTicker(housekeepingInterval)
	defer ticker.Stop()

	// Loop and wait for the ticker or cancellation.
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logger.Info("Performing regular housekeeping")
			housekeeping.Housekeep(logger)
		}
	}
}

// synchronizerMain is the entry point for the synchronizer command.
func synchronizerMain(_ *cobra.Command, _ []string) error {
	// Create a channel to track termination signals. We do this before creating
	// and starting other infrastructure so that we can ensure things terminate
	// smoothly, not mid-initialization.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)

	// Set up a logger on the standard error stream.
	logLevel := logging.LevelInfo
	if synchronizerConfiguration.logLevel != "" {
		if l, ok := logging.NameToLevel(synchronizerConfiguration.logLevel); !ok {
			return fmt.Errorf("invalid log level specified: %s", synchronizerConfiguration.logLevel)
		} else {
			logLevel = l
		}
	}
	logger := logging.NewLogger(logLevel, os.Stderr)

	// Set up regular housekeeping and defer its shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go housekeepRegularly(ctx, logger.Sublogger("housekeeping"))

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

	// Serve a synchronizer on standard input/output and monitor for its
	// termination.
	synchronizationTermination := make(chan error, 1)
	go func() {
		synchronizationTermination <- remote.ServeEndpoint(logger, stream)
	}()

	// Wait for termination from a signal or the synchronizer.
	select {
	case s := <-signalTermination:
		return fmt.Errorf("terminated by signal: %s", s)
	case err := <-synchronizationTermination:
		return fmt.Errorf("synchronization terminated: %w", err)
	}
}

// synchronizerCommand is the synchronizer command.
var synchronizerCommand = &cobra.Command{
	Use:          agent.CommandSynchronizer,
	Short:        "Run the agent in synchronizer mode",
	Args:         cmd.DisallowArguments,
	RunE:         synchronizerMain,
	SilenceUsage: true,
}

// synchronizerConfiguration stores configuration for the synchronizer command.
var synchronizerConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
	// logLevel indicates the log level to use.
	logLevel string
}

func init() {
	// Grab a handle for the command line flags.
	flags := synchronizerCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&synchronizerConfiguration.help, "help", "h", false, "Show help information")

	// Wire up logging flags.
	flags.StringVar(&synchronizerConfiguration.logLevel, agent.FlagLogLevel, "", "Set the log level")
}
