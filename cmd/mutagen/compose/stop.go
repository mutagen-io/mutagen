package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func stopMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("stop not yet implemented")
}

var stopCommand = &cobra.Command{
	Use:          "stop",
	Run:          stopMain,
	SilenceUsage: true,
}

var stopConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	stopCommand.SetHelpFunc(stopMain)

	// Grab a handle for the command line flags.
	flags := stopCommand.Flags()

	// Wire up stop command flags.
	flags.BoolVarP(&stopConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
