package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func downMain(_ *cobra.Command, arguments []string) {
	// TODO: Implement.
	fmt.Println("down not yet implemented")
}

var downCommand = &cobra.Command{
	Use:          "down",
	Run:          composeEntryPoint(downMain),
	SilenceUsage: true,
}

var downConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	downCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := downCommand.Flags()

	// Wire up down command flags.
	flags.BoolVarP(&downConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
