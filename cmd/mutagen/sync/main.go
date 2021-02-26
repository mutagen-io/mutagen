package sync

import (
	"github.com/spf13/cobra"
)

// syncMain is the entry point for the sync command.
func syncMain(command *cobra.Command, arguments []string) error {
	// If no commands were given, then print help information and bail. We don't
	// have to worry about warning about arguments being present here (which
	// would be incorrect usage) because arguments can't even reach this point
	// (they will be mistaken for subcommands and a error will be displayed).
	command.Help()

	// Success.
	return nil
}

// SyncCommand is the sync command.
var SyncCommand = &cobra.Command{
	Use:          "sync",
	Short:        "Create and manage file synchronization sessions",
	RunE:         syncMain,
	SilenceUsage: true,
}

// syncConfiguration stores configuration for the sync command.
var syncConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := SyncCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&syncConfiguration.help, "help", "h", false, "Show help information")

	// Register commands.
	SyncCommand.AddCommand(
		createCommand,
		listCommand,
		monitorCommand,
		flushCommand,
		pauseCommand,
		resumeCommand,
		resetCommand,
		terminateCommand,
	)
}
