package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func runMain(_ *cobra.Command, arguments []string) {
	// TODO: Implement.
	fmt.Println("run not yet implemented")
}

var runCommand = &cobra.Command{
	Use:          "run",
	Run:          composeEntryPoint(runMain),
	SilenceUsage: true,
}

var runConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	runCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := runCommand.Flags()

	// Wire up run command flags.
	flags.BoolVarP(&runConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
