package daemon

import (
	"os"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"

	"github.com/mutagen-io/mutagen/pkg/daemon"
)

// startMain is the entry point for the start command.
func startMain(_ *cobra.Command, _ []string) error {
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

// startCommand is the start command.
var startCommand = &cobra.Command{
	Use:          "start",
	Short:        "Start the Mutagen daemon if it's not already running",
	Args:         cmd.DisallowArguments,
	RunE:         startMain,
	SilenceUsage: true,
}

// startConfiguration stores configuration for the start command.
var startConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := startCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&startConfiguration.help, "help", "h", false, "Show help information")
}
