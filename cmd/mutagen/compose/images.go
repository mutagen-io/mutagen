package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func imagesMain(_ *cobra.Command, arguments []string) {
	// TODO: Implement.
	fmt.Println("images not yet implemented")
}

var imagesCommand = &cobra.Command{
	Use:          "images",
	Run:          composeEntryPoint(imagesMain),
	SilenceUsage: true,
}

var imagesConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	imagesCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := imagesCommand.Flags()

	// Wire up images command flags.
	flags.BoolVarP(&imagesConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
