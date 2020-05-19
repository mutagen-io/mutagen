package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func logsMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("logs not yet implemented")
}

var logsCommand = &cobra.Command{
	Use:          "logs",
	Run:          logsMain,
	SilenceUsage: true,
}

var logsConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	logsCommand.SetHelpFunc(logsMain)

	// Grab a handle for the command line flags.
	flags := logsCommand.Flags()

	// Wire up logs command flags.
	flags.BoolVarP(&logsConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
