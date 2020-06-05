package daemon

import (
	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"

	"github.com/mutagen-io/mutagen/pkg/daemon"
)

// unregisterMain is the entry point for the unregister command.
func unregisterMain(_ *cobra.Command, _ []string) error {
	return daemon.Unregister()
}

// unregisterCommand is the unregister command.
var unregisterCommand = &cobra.Command{
	Use:          "unregister",
	Short:        "Unregister automatic Mutagen daemon start-up [Experimental]",
	Args:         cmd.DisallowArguments,
	RunE:         unregisterMain,
	SilenceUsage: true,
}

// unregisterConfiguration stores configuration for the unregister command.
var unregisterConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := unregisterCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&unregisterConfiguration.help, "help", "h", false, "Show help information")
}
