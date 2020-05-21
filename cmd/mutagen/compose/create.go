package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func createMain(_ *cobra.Command, arguments []string) {
	// TODO: Implement.
	fmt.Println("create not yet implemented")
}

var createCommand = &cobra.Command{
	Use:          "create",
	Run:          composeEntryPoint(createMain),
	SilenceUsage: true,
}

var createConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	createCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := createCommand.Flags()

	// Wire up create command flags.
	flags.BoolVarP(&createConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
