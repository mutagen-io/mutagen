package cmd

import (
	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"

	// Explicitly import packages that need to register protocol handlers.
	_ "github.com/havoc-io/mutagen/pkg/protocols/docker"
	_ "github.com/havoc-io/mutagen/pkg/protocols/kubectl"
	_ "github.com/havoc-io/mutagen/pkg/protocols/local"
	_ "github.com/havoc-io/mutagen/pkg/protocols/ssh"
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

func NewMutagenCommand() *cobra.Command {

	var rootCommand = &cobra.Command{
		Use:   "mutagen",
		Short: "Mutagen provides simple, continuous, bi-directional file synchronization.",
		Run:   cmd.Mainify(rootMain),
	}

	var rootConfiguration struct {
		// help indicates whether or not help information should be shown for the
		// command.
		help bool
	}

	// Grab a handle for the command line flags.
	flags := rootCommand.Flags()

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
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
		createCommand,
		listCommand,
		monitorCommand,
		flushCommand,
		pauseCommand,
		resumeCommand,
		terminateCommand,
		daemonCommand,
		versionCommand,
		legalCommand,
		generateCommand,
	)

}
