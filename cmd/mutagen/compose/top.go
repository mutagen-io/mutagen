package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func topMain(_ *cobra.Command, arguments []string) {
	// TODO: Implement.
	fmt.Println("top not yet implemented")
}

var topCommand = &cobra.Command{
	Use:          "top",
	Run:          composeEntryPoint(topMain),
	SilenceUsage: true,
}

var topConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	topCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := topCommand.Flags()

	// Wire up top command flags.
	flags.BoolVarP(&topConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
