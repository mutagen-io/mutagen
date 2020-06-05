package forward

import (
	"github.com/spf13/cobra"
)

// forwardMain is the entry point for the forward command.
func forwardMain(command *cobra.Command, _ []string) error {
	// If no commands were given, then print help information and bail. We don't
	// have to worry about warning about arguments being present here (which
	// would be incorrect usage) because arguments can't even reach this point
	// (they will be mistaken for subcommands and a error will be displayed).
	command.Help()

	// Success.
	return nil
}

// ForwardCommand is the forward command.
var ForwardCommand = &cobra.Command{
	Use:          "forward",
	Short:        "Create and manage network forwarding sessions",
	RunE:         forwardMain,
	SilenceUsage: true,
}

// forwardConfiguration stores configuration for the forward command.
var forwardConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := ForwardCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&forwardConfiguration.help, "help", "h", false, "Show help information")

	// Register commands.
	ForwardCommand.AddCommand(
		createCommand,
		listCommand,
		monitorCommand,
		pauseCommand,
		resumeCommand,
		terminateCommand,
	)
}
