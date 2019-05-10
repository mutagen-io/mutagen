package main

import (
	"os"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/daemon"
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

	// Compute the path to the current executable.
	executablePath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "unable to determine executable path")
	}

	// Restart in the background.
	daemonProcess := &exec.Cmd{
		Path:        executablePath,
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
	Run:   cmd.Mainify(daemonStartMain),
}

var daemonStartConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := daemonStartCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&daemonStartConfiguration.help, "help", "h", false, "Show help information")
}
