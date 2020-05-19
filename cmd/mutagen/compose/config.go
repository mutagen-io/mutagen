package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func configMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("config not yet implemented")
}

var configCommand = &cobra.Command{
	Use:          "config",
	Run:          configMain,
	SilenceUsage: true,
}

var configConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	configCommand.SetHelpFunc(configMain)

	// Grab a handle for the command line flags.
	flags := configCommand.Flags()

	// Wire up config command flags.
	flags.BoolVarP(&configConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
