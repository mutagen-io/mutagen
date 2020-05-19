package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func buildMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("build not yet implemented")
}

var buildCommand = &cobra.Command{
	Use:          "build",
	Run:          buildMain,
	SilenceUsage: true,
}

var buildConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	buildCommand.SetHelpFunc(buildMain)

	// Grab a handle for the command line flags.
	flags := buildCommand.Flags()

	// Wire up build command flags.
	flags.BoolVarP(&buildConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
