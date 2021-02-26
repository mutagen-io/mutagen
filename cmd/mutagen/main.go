package main

import (
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/fatih/color"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"
	"github.com/mutagen-io/mutagen/cmd/mutagen/forward"
	"github.com/mutagen-io/mutagen/cmd/mutagen/project"
	"github.com/mutagen-io/mutagen/cmd/mutagen/sync"

	"github.com/mutagen-io/mutagen/pkg/prompting"
)

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
	Use:          "mutagen",
	Short:        "Fast file synchronization and network forwarding for remote development",
	RunE:         rootMain,
	SilenceUsage: true,
}

// rootConfiguration stores configuration for the root command.
var rootConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Disable alphabetical sorting of commands in help output. This is a global
	// setting that affects all Cobra command instances.
	cobra.EnableCommandSorting = false

	// Disable Cobra's use of mousetrap. This breaks daemon registration on
	// Windows because it tries to enforce that the CLI only be launched from
	// a console, which it's not when running automatically.
	cobra.MousetrapHelpText = ""

	// Grab a handle for the command line flags.
	flags := rootCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&rootConfiguration.help, "help", "h", false, "Show help information")

	// Register commands.
	rootCommand.AddCommand(
		sync.SyncCommand,
		forward.ForwardCommand,
		project.ProjectCommand,
		daemon.DaemonCommand,
		versionCommand,
		legalCommand,
		generateCommand,
	)

	// HACK If we're on Windows, enable color support for command usage and
	// error output by recursively replacing the output streams for Cobra
	// commands.
	if runtime.GOOS == "windows" {
		enableColorForCommand(rootCommand)
	}
}

// enableColorForCommand recursively enables colorized usage and error output
// for a command and all of its child commands.
func enableColorForCommand(command *cobra.Command) {
	// Enable color support for the command itself.
	command.SetOut(color.Output)
	command.SetErr(color.Error)

	// Recursively enable color support for child commands.
	for _, c := range command.Commands() {
		enableColorForCommand(c)
	}
}

func main() {
	// Check if a prompting environment is set. If so, treat this as a prompt
	// request. Prompting is sort of a special pseudo-command that's indicated
	// by the presence of an environment variable, and hence it has to be
	// handled in a bit of a special manner.
	if _, ok := os.LookupEnv(prompting.PrompterEnvironmentVariable); ok {
		if err := promptMain(os.Args[1:]); err != nil {
			cmd.Fatal(err)
		}
		return
	}

	// Handle terminal compatibility issues. If this call returns, it means that
	// we should proceed normally.
	cmd.HandleTerminalCompatibility()

	// Execute the root command.
	if err := rootCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
