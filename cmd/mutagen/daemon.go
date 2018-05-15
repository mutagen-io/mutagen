package main

import (
	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/pkg/daemon"
)

func daemonMain(command *cobra.Command, arguments []string) {
	// If no commands were given, then print help information and bail. We don't
	// have to worry about warning about arguments being present here (which
	// would be incorrect usage) because arguments can't even reach this point
	// (they will be mistaken for subcommands and a error will be displayed).
	command.Help()
}

var daemonCommand = &cobra.Command{
	Use:   "daemon",
	Short: "Controls the Mutagen daemon lifecycle",
	Run:   daemonMain,
}

var daemonConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := daemonCommand.Flags()
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
