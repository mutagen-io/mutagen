package compose

import (
	"github.com/spf13/cobra"
)

// pauseCommand is the pause command.
var pauseCommand = &cobra.Command{
	Use:                "pause",
	Run:                passthrough,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

// pauseConfiguration stores configuration for the pause command.
var pauseConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	pauseCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := pauseCommand.Flags()

	// Wire up pause command flags.
	flags.BoolVarP(&pauseConfiguration.help, "help", "h", false, "")
}
