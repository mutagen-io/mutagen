package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func eventsMain(_ *cobra.Command, arguments []string) {
	// TODO: Implement.
	fmt.Println("events not yet implemented")
}

var eventsCommand = &cobra.Command{
	Use:          "events",
	Run:          composeEntryPoint(eventsMain),
	SilenceUsage: true,
}

var eventsConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	eventsCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := eventsCommand.Flags()

	// Wire up events command flags.
	flags.BoolVarP(&eventsConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
