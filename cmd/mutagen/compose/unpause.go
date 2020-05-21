package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func unpauseMain(_ *cobra.Command, arguments []string) {
	// TODO: Implement.
	fmt.Println("unpause not yet implemented")
}

var unpauseCommand = &cobra.Command{
	Use:          "unpause",
	Run:          composeEntryPoint(unpauseMain),
	SilenceUsage: true,
}

var unpauseConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	unpauseCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := unpauseCommand.Flags()

	// Wire up unpause command flags.
	flags.BoolVarP(&unpauseConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
