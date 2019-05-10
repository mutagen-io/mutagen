package main

import (
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/daemon"
)

func daemonUnregisterMain(command *cobra.Command, arguments []string) error {
	// Validate arguments.
	if len(arguments) != 0 {
		return errors.New("unexpected arguments provided")
	}

	// Attempt deregistration.
	if err := daemon.Unregister(); err != nil {
		return err
	}

	// Success.
	return nil
}

var daemonUnregisterCommand = &cobra.Command{
	Use:   "unregister",
	Short: "Unregisters Mutagen as a per-user daemon",
	Run:   cmd.Mainify(daemonUnregisterMain),
}

var daemonUnregisterConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := daemonUnregisterCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&daemonUnregisterConfiguration.help, "help", "h", false, "Show help information")
}
