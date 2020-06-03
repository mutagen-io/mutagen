package main

import (
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/pkg/agent"
)

func installMain(_ *cobra.Command, _ []string) error {
	return errors.Wrap(agent.Install(), "installation error")
}

var installCommand = &cobra.Command{
	Use:          agent.ModeInstall,
	Short:        "Perform agent installation",
	Args:         cmd.DisallowArguments,
	RunE:         installMain,
	SilenceUsage: true,
}

var installConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := installCommand.Flags()

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&installConfiguration.help, "help", "h", false, "Show help information")
}
