package main

import (
	"context"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/daemon"
	daemonsvcpkg "github.com/havoc-io/mutagen/pkg/service/daemon"
)

func daemonStopMain(command *cobra.Command, arguments []string) error {
	// Validate arguments.
	if len(arguments) != 0 {
		return errors.New("unexpected arguments provided")
	}

	// If the daemon is registered with the system, it may have a different stop
	// mechanism, so see if the system should handle it.
	if handled, err := daemon.RegisteredStop(); err != nil {
		return errors.Wrap(err, "unable to stop daemon using system mechanism")
	} else if handled {
		return nil
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := createDaemonClientConnection()
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Create a daemon service client.
	daemonService := daemonsvcpkg.NewDaemonClient(daemonConnection)

	// Invoke shutdown. We don't check the response or error, because the daemon
	// may terminate before it has a chance to send the response.
	daemonService.Terminate(context.Background(), &daemonsvcpkg.TerminateRequest{})

	// Success.
	return nil
}

var daemonStopCommand = &cobra.Command{
	Use:   "stop",
	Short: "Stops the Mutagen daemon if it's running",
	Run:   cmd.Mainify(daemonStopMain),
}

var daemonStopConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := daemonStopCommand.Flags()
	flags.BoolVarP(&daemonStopConfiguration.help, "help", "h", false, "Show help information")
}
