package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func runMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("run not yet implemented")
}

var runCommand = &cobra.Command{
	Use:          "run",
	Run:          runMain,
	SilenceUsage: true,
}

var runConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	runCommand.SetHelpFunc(runMain)

	// Grab a handle for the command line flags.
	flags := runCommand.Flags()

	// Wire up run command flags.
	flags.BoolVarP(&runConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
