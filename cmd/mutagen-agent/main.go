package main

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	// Disable logging flags for the standard logger (used by the logging
	// infrastructure) so that agent standard error output (which is logged by
	// the daemon) isn't doubly-prefixed.
	log.SetFlags(0)
}

// rootMain is the entry point for the root command.
func rootMain(command *cobra.Command, _ []string) error {
	// If no commands were given, then print help information and bail. We don't
	// have to worry about warning about arguments being present here (which
	// would be incorrect usage) because arguments can't even reach this point
	// (they will be mistaken for subcommands and a error will be displayed).
	command.Help()

	// Success.
	return nil
}

// rootCommand is the root command.
var rootCommand = &cobra.Command{
	Use:          "mutagen-agent",
	Short:        "The Mutagen agent should not be invoked by human beings",
	RunE:         rootMain,
	SilenceUsage: true,
}

// rootConfiguration stores configuration for the root command.
var rootConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Disable Cobra's command sorting behavior. By default, it sorts commands
	// alphabetically in the help output.
	cobra.EnableCommandSorting = false

	// Disable Cobra's use of mousetrap. This breaks daemon registration on
	// Windows because it tries to enforce that the CLI only be launched from
	// a console, which it's not when running automatically.
	cobra.MousetrapHelpText = ""

	// Grab a handle for the command line flags.
	flags := rootCommand.Flags()

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&rootConfiguration.help, "help", "h", false, "Show help information")

	// Register commands. We do this here (rather than in individual init
	// functions) so that we can control the order.
	rootCommand.AddCommand(
		installCommand,
		synchronizerCommand,
		forwarderCommand,
		versionCommand,
		legalCommand,
	)
}

func main() {
	// Execute the root command.
	if err := rootCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
