package main

import (
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"

	"github.com/mutagen-io/mutagen/pkg/mutagen"
)

// rootMain is the entry point for the root command.
func rootMain(command *cobra.Command, _ []string) error {
	// Set up signal handling.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)

	// Wait for termination.
	<-signalTermination

	// Success.
	return nil
}

// rootCommand is the root command.
var rootCommand = &cobra.Command{
	Use:          "mutagen-sidecar",
	Version:      mutagen.Version,
	Short:        "Sidecar entry point for creating and controlling Mutagen sessions",
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

	// Disable Cobra's use of mousetrap. This is primarily for consistency with
	// the main CLI, as it's not necessary for the sidecar.
	cobra.MousetrapHelpText = ""

	// Set the template used by the version flag.
	rootCommand.SetVersionTemplate("Mutagen sidecar version {{ .Version }}\n")

	// Grab a handle for the command line flags.
	flags := rootCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&rootConfiguration.help, "help", "h", false, "Show help information")

	// Register commands. We do this here (rather than in individual init
	// functions) so that we can control the order.
	rootCommand.AddCommand(
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
