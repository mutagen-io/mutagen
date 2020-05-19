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
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	portCommand.SetHelpFunc(portMain)

	// Grab a handle for the command line flags.
	flags := portCommand.Flags()

	// Wire up port command flags.
	flags.BoolVarP(&portConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
