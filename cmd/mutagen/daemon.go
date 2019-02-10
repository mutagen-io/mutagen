package main

import (
	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/daemon"
)

func daemonMain(command *cobra.Command, arguments []string) error {
	// If no commands were given, then print help information and bail. We don't
	// have to worry about warning about arguments being present here (which
	// would be incorrect usage) because arguments can't even reach this point
	// (they will be mistaken for subcommands and a error will be displayed).
	command.Help()

	// Success.
	return nil
}

var daemonCommand = &cobra.Command{
	Use:   "daemon",
	Short: "Controls the Mutagen daemon lifecycle",
	Run:   cmd.Mainify(daemonMain),
}

var daemonConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := daemonCommand.Flags()

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&daemonConfiguration.help, "help", "h", false, "Show help information")

	// Register commands. We do this here (rather than in individual init
	// functions) so that we can control the order. If registration isn't
	// supported on the platform, then we exclude those commands. For some
	// reason, AddCommand can't be invoked twice, so we can't add these commands
	// conditionally later.
	if daemon.RegistrationSupported {
		daemonCommand.AddCommand(
			daemonRunCommand,
			daemonStartCommand,
			daemonStopCommand,
			daemonRegisterCommand,
			daemonUnregisterCommand,
		)
	} else {
		daemonCommand.AddCommand(
			daemonRunCommand,
			daemonStartCommand,
			daemonStopCommand,
		)
	}
}
