package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"

	"github.com/mutagen-io/mutagen/pkg/mutagen"
)

// versionMain is the entry point for the version command.
func versionMain(_ *cobra.Command, _ []string) error {
	// Print version information.
	fmt.Println(mutagen.Version)

	// Success.
	return nil
}

// versionCommand is the version command.
var versionCommand = &cobra.Command{
	Use:          "version",
	Short:        "Show version information",
	Args:         cmd.DisallowArguments,
	RunE:         versionMain,
	SilenceUsage: true,
}

// versionConfiguration stores configuration for the version command.
var versionConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := versionCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&versionConfiguration.help, "help", "h", false, "Show help information")
}
