package main

import (
	"os"
	"os/signal"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/mutagen"
	"github.com/havoc-io/mutagen/pkg/session"
)

func endpointMain(command *cobra.Command, arguments []string) error {
	// Create a connection on standard input/output.
	connection := newStdioConnection()

	// Perform a handshake.
	if err := mutagen.SendVersion(connection); err != nil {
		return errors.Wrap(err, "unable to transmit version")
	}

	// Serve an endpoint on standard input/output and monitor for its
	// termination.
	endpointTermination := make(chan error, 1)
	go func() {
		endpointTermination <- session.ServeEndpoint(connection)
	}()

	// Wait for termination from a signal or the endpoint.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)
	select {
	case sig := <-signalTermination:
		return errors.Errorf("terminated by signal: %s", sig)
	case err := <-endpointTermination:
		return errors.Wrap(err, "endpoint terminated")
	}
}

var endpointCommand = &cobra.Command{
	Use:   "endpoint",
	Short: "Run the agent in endpoint mode",
	Run:   cmd.Mainify(endpointMain),
}

var endpointConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := endpointCommand.Flags()
	flags.BoolVarP(&endpointConfiguration.help, "help", "h", false, "Show help information")
}
