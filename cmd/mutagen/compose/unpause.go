package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func unpauseMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("unpause not yet implemented")
}

var unpauseCommand = &cobra.Command{
	Use:          "unpause",
	Run:          unpauseMain,
	SilenceUsage: true,
}

var unpauseConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	unpauseCommand.SetHelpFunc(unpauseMain)

	// Grab a handle for the command line flags.
	flags := unpauseCommand.Flags()

	// Wire up unpause command flags.
	flags.BoolVarP(&unpauseConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
