package daemon

import (
	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/pkg/daemon"
)

func rootMain(command *cobra.Command, arguments []string) error {
	// If no commands were given, then print help information and bail. We don't
	// have to worry about warning about arguments being present here (which
	// would be incorrect usage) because arguments can't even reach this point
	// (they will be mistaken for subcommands and a error will be displayed).
	command.Help()

	// Success.
	return nil
}

var RootCommand = &cobra.Command{
	Use:          "daemon",
	Short:        "Control the lifecycle of the Mutagen daemon",
	RunE:         rootMain,
	SilenceUsage: true,
}

var rootConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := RootCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&rootConfiguration.help, "help", "h", false, "Show help information")

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
	RootCommand.AddCommand(supportedCommands...)
}
