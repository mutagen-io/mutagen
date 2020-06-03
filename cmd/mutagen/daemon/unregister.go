package daemon

import (
	"github.com/spf13/cobra"

	"github.com/fatih/color"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/pkg/daemon"
)

func unregisterMain(_ *cobra.Command, _ []string) error {
	return daemon.Unregister()
}

var unregisterCommand = &cobra.Command{
	Use:          "unregister",
	Short:        "Unregister automatic Mutagen daemon start-up",
	Args:         cmd.DisallowArguments,
	RunE:         unregisterMain,
	SilenceUsage: true,
}

var unregisterConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Mark the command as experimental.
	unregisterCommand.Short = unregisterCommand.Short + color.YellowString(" [Experimental]")

	// Grab a handle for the command line flags.
	flags := unregisterCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&unregisterConfiguration.help, "help", "h", false, "Show help information")
}
