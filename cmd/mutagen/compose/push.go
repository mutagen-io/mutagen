package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func pushMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("push not yet implemented")
}

var pushCommand = &cobra.Command{
	Use:          "push",
	Run:          pushMain,
	SilenceUsage: true,
}

var pushConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	pushCommand.SetHelpFunc(pushMain)

	// Grab a handle for the command line flags.
	flags := pushCommand.Flags()

	// Wire up push command flags.
	flags.BoolVarP(&pushConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
