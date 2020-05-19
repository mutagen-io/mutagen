package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func killMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("kill not yet implemented")
}

var killCommand = &cobra.Command{
	Use:          "kill",
	Run:          killMain,
	SilenceUsage: true,
}

var killConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	killCommand.SetHelpFunc(killMain)

	// Grab a handle for the command line flags.
	flags := killCommand.Flags()

	// Wire up kill command flags.
	flags.BoolVarP(&killConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
