package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
)

func legalMain(_ *cobra.Command, _ []string) error {
	// Print legal information.
	fmt.Println(mutagen.LegalNotice)

	// Success.
	return nil
}

var legalCommand = &cobra.Command{
	Use:          "legal",
	Short:        "Show legal information",
	Args:         cmd.DisallowArguments,
	RunE:         legalMain,
	SilenceUsage: true,
}

var legalConfiguration struct {
	// help indicates whether or not to show help information and exit.
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
