package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"

	"github.com/mutagen-io/mutagen/pkg/agent"
)

// installMain is the entry point for the install command.
func installMain(_ *cobra.Command, _ []string) error {
	// Perform the installation.
	if err := agent.Install(); err != nil {
		return fmt.Errorf("installation error: %w", err)
	}

	// Success.
	return nil
}

// installCommand is the install command.
var installCommand = &cobra.Command{
	Use:          agent.CommandInstall,
	Short:        "Perform agent installation",
	Args:         cmd.DisallowArguments,
	RunE:         installMain,
	SilenceUsage: true,
}

// installConfiguration stores configuration for the install command.
var installConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := installCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&installConfiguration.help, "help", "h", false, "Show help information")
}
