package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func logsMain(_ *cobra.Command, arguments []string) {
	// TODO: Implement.
	fmt.Println("logs not yet implemented")
}

var logsCommand = &cobra.Command{
	Use:          "logs",
	Run:          composeEntryPoint(logsMain),
	SilenceUsage: true,
}

var logsConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	logsCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := logsCommand.Flags()

	// Wire up logs command flags.
	flags.BoolVarP(&logsConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
