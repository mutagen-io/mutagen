package compose

import (
	"fmt"

	"github.com/spf13/cobra"
)

func imagesMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// TODO: Implement.
	fmt.Println("images not yet implemented")
}

var imagesCommand = &cobra.Command{
	Use:          "images",
	Run:          imagesMain,
	SilenceUsage: true,
}

var imagesConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	imagesCommand.SetHelpFunc(imagesMain)

	// Grab a handle for the command line flags.
	flags := imagesCommand.Flags()

	// Wire up images command flags.
	flags.BoolVarP(&imagesConfiguration.help, "help", "h", false, "")
	// TODO: Wire up remaining flags.
}
