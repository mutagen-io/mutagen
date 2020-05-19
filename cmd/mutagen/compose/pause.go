package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func pauseMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("pause not yet implemented")
}

var pauseCommand = &cobra.Command{
	Use:          "pause",
	Run:          pauseMain,
	SilenceUsage: true,
}

var pauseConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	pauseCommand.SetHelpFunc(pauseMain)

	// Grab a handle for the command line flags.
	flags := pauseCommand.Flags()

	// Wire up pause command flags.
	flags.BoolVarP(&pauseConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
