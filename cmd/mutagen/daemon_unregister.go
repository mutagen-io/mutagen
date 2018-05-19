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
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := daemonUnregisterCommand.Flags()
	flags.BoolVarP(&daemonUnregisterConfiguration.help, "help", "h", false, "Show help information")
}
