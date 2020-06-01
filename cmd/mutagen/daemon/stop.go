package daemon

import (
	"context"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/pkg/daemon"
	daemonsvc "github.com/mutagen-io/mutagen/pkg/service/daemon"
)

func stopMain(command *cobra.Command, arguments []string) error {
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
	daemonConnection, err := Connect(false, false)
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Create a daemon service client.
	daemonService := daemonsvc.NewDaemonClient(daemonConnection)

	// Invoke shutdown. We don't check the response or error, because the daemon
	// may terminate before it has a chance to send the response.
	daemonService.Terminate(context.Background(), &daemonsvc.TerminateRequest{})

	// Success.
	return nil
}

var stopCommand = &cobra.Command{
	Use:          "stop",
	Short:        "Stop the Mutagen daemon if it's running",
	RunE:         stopMain,
	SilenceUsage: true,
}

var stopConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := stopCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&stopConfiguration.help, "help", "h", false, "Show help information")
}
