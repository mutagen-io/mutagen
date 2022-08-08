package project

import (
	"github.com/spf13/cobra"

	// Explicitly import packages that need to register protocol handlers.
	_ "github.com/mutagen-io/mutagen/pkg/forwarding/protocols/docker"
	_ "github.com/mutagen-io/mutagen/pkg/forwarding/protocols/local"
	_ "github.com/mutagen-io/mutagen/pkg/forwarding/protocols/ssh"
)

// projectMain is the entry point for the project command.
func projectMain(command *cobra.Command, _ []string) error {
	// If no commands were given, then print help information and bail. We don't
	// have to worry about warning about arguments being present here (which
	// would be incorrect usage) because arguments can't even reach this point
	// (they will be mistaken for subcommands and a error will be displayed).
	command.Help()

	// Success.
	return nil
}

// ProjectCommand is the project command.
var ProjectCommand = &cobra.Command{
	Use:          "project",
	Short:        "Orchestrate sessions for a Mutagen project [Experimental]",
	RunE:         projectMain,
	SilenceUsage: true,
}

// projectConfiguration stores configuration for the project command.
var projectConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := ProjectCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&projectConfiguration.help, "help", "h", false, "Show help information")

	// Register commands.
	ProjectCommand.AddCommand(
		startCommand,
		runCommand,
		listCommand,
		flushCommand,
		pauseCommand,
		resumeCommand,
		resetCommand,
		terminateCommand,
	)
}
