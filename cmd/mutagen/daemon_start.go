package main

import (
	"os/exec"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/pkg/daemon"
	"github.com/havoc-io/mutagen/pkg/process"
)

func daemonStartMain(command *cobra.Command, arguments []string) error {
	// Validate arguments.
	if len(arguments) != 0 {
		return errors.New("unexpected arguments provided")
	}

	// If the daemon is registered with the system, it may have a different
	// start mechanism, so see if the system should handle it.
	if handled, err := daemon.RegisteredStart(); err != nil {
		return errors.Wrap(err, "unable to start daemon using system mechanism")
	} else if handled {
		return nil
	}

	// Restart in the background.
	daemonProcess := &exec.Cmd{
		Path:        process.Current.ExecutablePath,
		Args:        []string{"mutagen", "daemon", "run"},
		SysProcAttr: daemonProcessAttributes,
	}
	if err := daemonProcess.Start(); err != nil {
		return errors.Wrap(err, "unable to fork daemon")
	}

	// Success.
	return nil
}

var daemonStartCommand = &cobra.Command{
	Use:   "start",
	Short: "Starts the Mutagen daemon if it's not already running",
	Run:   mainify(daemonStartMain),
}

var daemonStartConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := daemonStartCommand.Flags()
	flags.BoolVarP(&daemonStartConfiguration.help, "help", "h", false, "Show help information")
}
