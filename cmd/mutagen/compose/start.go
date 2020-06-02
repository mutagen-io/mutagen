package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func startMain(_ *cobra.Command, arguments []string) error {
	// TODO: Implement.
	fmt.Println("start not yet implemented")
	return nil
}

var startCommand = &cobra.Command{
	Use:          "start",
	RunE:         composeEntryPointE(startMain),
	SilenceUsage: true,
}

var startConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	startCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := startCommand.Flags()

	// Wire up start command flags.
	flags.BoolVarP(&startConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
