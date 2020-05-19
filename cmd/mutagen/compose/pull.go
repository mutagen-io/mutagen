package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func pullMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("pull not yet implemented")
}

var pullCommand = &cobra.Command{
	Use:          "pull",
	Run:          pullMain,
	SilenceUsage: true,
}

var pullConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	pullCommand.SetHelpFunc(pullMain)

	// Grab a handle for the command line flags.
	flags := pullCommand.Flags()

	// Wire up pull command flags.
	flags.BoolVarP(&pullConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
