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
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := installCommand.Flags()
	flags.BoolVarP(&installConfiguration.help, "help", "h", false, "Show help information")
}
