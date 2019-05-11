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

	// Connect to the daemon and defer closure of the connection. We avoid
	// version compatibility checks since they would remove the ability to
	// terminate an incompatible daemon. This is fine since the daemon service
	// portion of the daemon API is stable.
	daemonConnection, err := createDaemonClientConnection(false)
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
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := daemonStopCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&daemonStopConfiguration.help, "help", "h", false, "Show help information")
}
