package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func psMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("ps not yet implemented")
}

var psCommand = &cobra.Command{
	Use:          "ps",
	Run:          psMain,
	SilenceUsage: true,
}

var psConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	psCommand.SetHelpFunc(psMain)

	// Grab a handle for the command line flags.
	flags := psCommand.Flags()

	// Wire up ps command flags.
	flags.BoolVarP(&psConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
