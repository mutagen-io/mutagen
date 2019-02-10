package main

import (
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/agent"
)

func installMain(command *cobra.Command, arguments []string) error {
	return errors.Wrap(agent.Install(), "installation error")
}

var installCommand = &cobra.Command{
	Use:   agent.ModeInstall,
	Short: "Perform agent installation",
	Run:   cmd.Mainify(installMain),
}

var installConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := installCommand.Flags()

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&installConfiguration.help, "help", "h", false, "Show help information")
}
