package daemon

import (
	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"

	"github.com/mutagen-io/mutagen/pkg/daemon"
)

// registerMain is the entry point for the register command.
func registerMain(_ *cobra.Command, _ []string) error {
	return daemon.Register()
}

// registerCommand is the register command.
var registerCommand = &cobra.Command{
	Use:          "register",
	Short:        "Register the Mutagen daemon to start automatically on login [Experimental]",
	Args:         cmd.DisallowArguments,
	RunE:         registerMain,
	SilenceUsage: true,
}

// registerConfiguration stores configuration for the register command.
var registerConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := registerCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&registerConfiguration.help, "help", "h", false, "Show help information")
}
