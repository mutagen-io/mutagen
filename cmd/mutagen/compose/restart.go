package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func restartMain(_ *cobra.Command, arguments []string) {
	// TODO: Implement.
	fmt.Println("restart not yet implemented")
}

var restartCommand = &cobra.Command{
	Use:          "restart",
	Run:          composeEntryPoint(restartMain),
	SilenceUsage: true,
}

var restartConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
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
	// TODO: Wire up remaining flags.
}
