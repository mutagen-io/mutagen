package compose

import (
	"github.com/spf13/cobra"
)

// restartCommand is the restart command.
var restartCommand = &cobra.Command{
	Use:                "restart",
	Run:                passthrough,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

// restartConfiguration stores configuration for the restart command.
var restartConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// timeout stores the value of the -t/--timeout flag.
	timeout string
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	restartCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := restartCommand.Flags()

	// Wire up restart command flags.
	flags.BoolVarP(&restartConfiguration.help, "help", "h", false, "")
	flags.StringVarP(&restartConfiguration.timeout, "timeout", "t", "", "")
}
