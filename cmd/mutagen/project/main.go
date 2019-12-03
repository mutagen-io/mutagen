package project

import (
	"github.com/spf13/cobra"

	"github.com/fatih/color"

	// Explicitly import packages that need to register protocol handlers.
	_ "github.com/mutagen-io/mutagen/pkg/forwarding/protocols/docker"
	_ "github.com/mutagen-io/mutagen/pkg/forwarding/protocols/local"
	_ "github.com/mutagen-io/mutagen/pkg/forwarding/protocols/ssh"
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

var RootCommand = &cobra.Command{
	Use:          "project",
	Short:        "Orchestrate sessions for a project",
	RunE:         rootMain,
	SilenceUsage: true,
}

var rootConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
}

func init() {
	// Mark the command as experimental.
	RootCommand.Short = RootCommand.Short + color.YellowString(" [Beta]")

	// Grab a handle for the command line flags.
	flags := RootCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&rootConfiguration.help, "help", "h", false, "Show help information")

	// Register commands.
	RootCommand.AddCommand(
		startCommand,
		listCommand,
		flushCommand,
		pauseCommand,
		resumeCommand,
		terminateCommand,
	)
}
