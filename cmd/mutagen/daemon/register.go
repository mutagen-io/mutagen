package daemon

import (
	"github.com/spf13/cobra"

	"github.com/fatih/color"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/pkg/daemon"
)

func registerMain(_ *cobra.Command, _ []string) error {
	return daemon.Register()
}

var registerCommand = &cobra.Command{
	Use:          "register",
	Short:        "Register the Mutagen daemon to start automatically on login",
	Args:         cmd.DisallowArguments,
	RunE:         registerMain,
	SilenceUsage: true,
}

var registerConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Mark the command as experimental.
	registerCommand.Short = registerCommand.Short + color.YellowString(" [Experimental]")

	// Grab a handle for the command line flags.
	flags := registerCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&registerConfiguration.help, "help", "h", false, "Show help information")
}
