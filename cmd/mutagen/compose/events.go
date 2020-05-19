package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func eventsMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("events not yet implemented")
}

var eventsCommand = &cobra.Command{
	Use:          "events",
	Run:          eventsMain,
	SilenceUsage: true,
}

var eventsConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	eventsCommand.SetHelpFunc(eventsMain)

	// Grab a handle for the command line flags.
	flags := eventsCommand.Flags()

	// Wire up events command flags.
	flags.BoolVarP(&eventsConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
