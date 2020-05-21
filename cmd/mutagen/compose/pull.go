package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func pullMain(_ *cobra.Command, arguments []string) {
	// TODO: Implement.
	fmt.Println("pull not yet implemented")
}

var pullCommand = &cobra.Command{
	Use:          "pull",
	Run:          composeEntryPoint(pullMain),
	SilenceUsage: true,
}

var pullConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	pullCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := pullCommand.Flags()

	// Wire up pull command flags.
	flags.BoolVarP(&pullConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
