package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/housekeeping"
	"github.com/havoc-io/mutagen/pkg/session/endpoint/remote"
)

const (
	// housekeepingInterval is the interval at which housekeeping will be
	// invoked by the agent.
	housekeepingInterval = 24 * time.Hour
)

func housekeepRegularly(context context.Context) {
	// Perform an initial housekeeping operation since the ticker won't fire
	// straight away.
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
			housekeeping.Housekeep()
		}
	}
}

func endpointMain(command *cobra.Command, arguments []string) error {
	// Create a channel to track termination signals. We do this before creating
	// and starting other infrastructure so that we can ensure things terminate
	// smoothly, not mid-initialization.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)

	// Set up regular housekeeping and defer its shutdown.
	housekeepingContext, housekeepingCancel := context.WithCancel(context.Background())
	defer housekeepingCancel()
	go housekeepRegularly(housekeepingContext)

	// Create a connection on standard input/output.
	connection := newStdioConnection()

	// Serve an endpoint on standard input/output and monitor for its
	// termination.
	endpointTermination := make(chan error, 1)
	go func() {
		endpointTermination <- remote.ServeEndpoint(connection)
	}()

	// Wait for termination from a signal or the endpoint.
	select {
	case sig := <-signalTermination:
		return errors.Errorf("terminated by signal: %s", sig)
	case err := <-endpointTermination:
		return errors.Wrap(err, "endpoint terminated")
	}
}

var endpointCommand = &cobra.Command{
	Use:   agent.ModeEndpoint,
	Short: "Run the agent in endpoint mode",
	Run:   cmd.Mainify(endpointMain),
}

var endpointConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := endpointCommand.Flags()

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&endpointConfiguration.help, "help", "h", false, "Show help information")
}
