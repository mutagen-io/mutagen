package daemon

import (
	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/pkg/daemon"
)

// daemonMain is the entry point for the daemon command.
func daemonMain(command *cobra.Command, _ []string) error {
	// If no commands were given, then print help information and bail. We don't
	// have to worry about warning about arguments being present here (which
	// would be incorrect usage) because arguments can't even reach this point
	// (they will be mistaken for subcommands and a error will be displayed).
	command.Help()

	// Success.
	return nil
}

// DaemonCommand is the daemon command.
var DaemonCommand = &cobra.Command{
	Use:          "daemon",
	Short:        "Control the lifecycle of the Mutagen daemon",
	RunE:         daemonMain,
	SilenceUsage: true,
}

// daemonConfiguration stores configuration for the daemon command.
var daemonConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := DaemonCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&daemonConfiguration.help, "help", "h", false, "Show help information")

	// Compute supported commands. We have to do this in advance since
	// AddCommand can't be invoked twice.
	supportedCommands := []*cobra.Command{
		runCommand,
		startCommand,
		stopCommand,
	}
	if daemon.RegistrationSupported {
		supportedCommands = append(supportedCommands,
			registerCommand,
			unregisterCommand,
		)
	}

	// Register commands.
	DaemonCommand.AddCommand(supportedCommands...)
}
