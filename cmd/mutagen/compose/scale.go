package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func scaleMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("scale not yet implemented")
}

var scaleCommand = &cobra.Command{
	Use:          "scale",
	Run:          scaleMain,
	SilenceUsage: true,
}

var scaleConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	scaleCommand.SetHelpFunc(scaleMain)

	// Grab a handle for the command line flags.
	flags := scaleCommand.Flags()

	// Wire up scale command flags.
	flags.BoolVarP(&scaleConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
