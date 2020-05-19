package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func topMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("top not yet implemented")
}

var topCommand = &cobra.Command{
	Use:          "top",
	Run:          topMain,
	SilenceUsage: true,
}

var topConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	topCommand.SetHelpFunc(topMain)

	// Grab a handle for the command line flags.
	flags := topCommand.Flags()

	// Wire up top command flags.
	flags.BoolVarP(&topConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
