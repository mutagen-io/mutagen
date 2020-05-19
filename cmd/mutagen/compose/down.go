package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func downMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("down not yet implemented")
}

var downCommand = &cobra.Command{
	Use:          "down",
	Run:          downMain,
	SilenceUsage: true,
}

var downConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	downCommand.SetHelpFunc(downMain)

	// Grab a handle for the command line flags.
	flags := downCommand.Flags()

	// Wire up down command flags.
	flags.BoolVarP(&downConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
