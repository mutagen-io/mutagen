package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func createMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("create not yet implemented")
}

var createCommand = &cobra.Command{
	Use:          "create",
	Run:          createMain,
	SilenceUsage: true,
}

var createConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	createCommand.SetHelpFunc(createMain)

	// Grab a handle for the command line flags.
	flags := createCommand.Flags()

	// Wire up create command flags.
	flags.BoolVarP(&createConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
