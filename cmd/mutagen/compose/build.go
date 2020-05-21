package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func buildMain(_ *cobra.Command, arguments []string) {
	// TODO: Implement.
	fmt.Println("build not yet implemented")
}

var buildCommand = &cobra.Command{
	Use:          "build",
	Run:          composeEntryPoint(buildMain),
	SilenceUsage: true,
}

var buildConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	buildCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := buildCommand.Flags()

	// Wire up build command flags.
	flags.BoolVarP(&buildConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
