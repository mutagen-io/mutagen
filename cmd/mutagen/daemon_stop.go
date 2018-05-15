package main

import (
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/daemon"
)

func daemonStopMain(command *cobra.Command, arguments []string) {
	// Validate arguments.
	if len(arguments) != 0 {
		cmd.Fatal(errors.New("unexpected arguments provided"))
	}

	// If the daemon is registered with the system, it may have a different stop
	// mechanism, so see if the system should handle it.
	if handled, err := daemon.RegisteredStop(); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to stop daemon using system mechanism"))
	} else if handled {
		return
	}

	// Create a daemon client and defer its closure.
	daemonClient, err := createDaemonClient()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create daemon client"))
	}
	defer daemonClient.Close()

	// Invoke termination.
	stream, err := daemonClient.Invoke(daemon.MethodTerminate)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to invoke daemon termination"))
	}
	stream.Close()
}

var daemonStopCommand = &cobra.Command{
	Use:   "stop",
	Short: "Stops the Mutagen daemon if it's running",
	Run:   daemonStopMain,
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
