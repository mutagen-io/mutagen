package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/mutagen"
)

func legalMain(command *cobra.Command, arguments []string) error {
	// Print legal information.
	fmt.Println(mutagen.LegalNotice)

	// Success.
	return nil
}

var legalCommand = &cobra.Command{
	Use:   "legal",
	Short: "Show legal information",
	Run:   cmd.Mainify(legalMain),
}

var legalConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := legalCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&legalConfiguration.help, "help", "h", false, "Show help information")
}
