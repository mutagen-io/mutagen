package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func portMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("port not yet implemented")
}

var portCommand = &cobra.Command{
	Use:          "port",
	Run:          portMain,
	SilenceUsage: true,
}

var portConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	portCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := portCommand.Flags()

	// Wire up port command flags.
	flags.BoolVarP(&portConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
