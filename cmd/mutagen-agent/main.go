package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/session"
)

func rootMain(command *cobra.Command, arguments []string) error {
	// If no commands were given, then print help information and bail. We don't
	// have to worry about warning about arguments being present here (which
	// would be incorrect usage) because arguments can't even reach this point
	// (they will be mistaken for subcommands and a error will be displayed).
	command.Help()

	// Success.
	return nil
}

var rootCommand = &cobra.Command{
	Use:   "mutagen-agent",
	Short: "The Mutagen agent should not be invoked by human beings.",
	Run:   cmd.Mainify(rootMain),
}

var rootConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := rootCommand.Flags()
	flags.BoolVarP(&rootConfiguration.help, "help", "h", false, "Show help information")

	// Disable Cobra's command sorting behavior. By default, it sorts commands
	// alphabetically in the help output.
	cobra.EnableCommandSorting = false

	// Disable Cobra's use of mousetrap. This breaks daemon registration on
	// Windows because it tries to enforce that the CLI only be launched from
	// a console, which it's not when running automatically.
	cobra.MousetrapHelpText = ""

	// Register commands. We do this here (rather than in individual init
	// functions) so that we can control the order.
	rootCommand.AddCommand(
		installCommand,
		endpointCommand,
		versionCommand,
		legalCommand,
	)
}

func main() {
	// Perform housekeeping.
	agent.Housekeep()
	session.HousekeepCaches()
	session.HousekeepStaging()

	// Execute the root command.
	if err := rootCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
