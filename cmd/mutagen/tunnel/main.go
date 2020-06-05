package tunnel

import (
	"github.com/spf13/cobra"
)

// tunnelMain is the entry point for the tunnel command.
func tunnelMain(command *cobra.Command, _ []string) error {
	// If no commands were given, then print help information and bail. We don't
	// have to worry about warning about arguments being present here (which
	// would be incorrect usage) because arguments can't even reach this point
	// (they will be mistaken for subcommands and a error will be displayed).
	command.Help()

	// Success.
	return nil
}

// TunnelCommand is the tunnel command.
var TunnelCommand = &cobra.Command{
	Use:          "tunnel",
	Short:        "Create and manage tunnels [Experimental]",
	RunE:         tunnelMain,
	SilenceUsage: true,
}

// tunnelConfiguration stores configuration for the tunnel command.
var tunnelConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := TunnelCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&tunnelConfiguration.help, "help", "h", false, "Show help information")

	// Register commands.
	TunnelCommand.AddCommand(
		createCommand,
		listCommand,
		monitorCommand,
		pauseCommand,
		resumeCommand,
		terminateCommand,
		hostCommand,
	)
}
