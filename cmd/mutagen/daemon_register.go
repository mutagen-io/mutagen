package main

import (
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/daemon"
)

func daemonRegisterMain(command *cobra.Command, arguments []string) error {
	// Validate arguments.
	if len(arguments) != 0 {
		return errors.New("unexpected arguments provided")
	}

	// Attempt registration.
	if err := daemon.Register(); err != nil {
		return err
	}

	// Success.
	return nil
}

var daemonRegisterCommand = &cobra.Command{
	Use:   "register",
	Short: "Registers Mutagen to start as a per-user daemon on login",
	Run:   cmd.Mainify(daemonRegisterMain),
}

var daemonRegisterConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := daemonRegisterCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&daemonRegisterConfiguration.help, "help", "h", false, "Show help information")
}
