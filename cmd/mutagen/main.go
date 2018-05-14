package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/environment"
	"github.com/havoc-io/mutagen/pkg/mutagen"
	"github.com/havoc-io/mutagen/pkg/ssh"
)

func rootMain(command *cobra.Command, arguments []string) {
	// Sanity check configuration.
	if rootConfiguration.version && rootConfiguration.legal {
		cmd.Fatal(errors.New("version and legal information requested simultaneously"))
	}

	// Print version information, if requested.
	if rootConfiguration.version {
		fmt.Println(mutagen.Version)
		return
	}

	// Print legal information, if requested.
	if rootConfiguration.legal {
		fmt.Print(mutagen.LegalNotice)
		return
	}

	// If no flags were set, then print help information and bail. We don't have
	// to worry about warning about arguments being present here (which would
	// be incorrect usage) because arguments can't even reach this point (they
	// will be mistaken for subcommands and a error will be displayed).
	command.Help()
}

var rootCommand = &cobra.Command{
	Use:   "mutagen",
	Short: "Mutagen provides simple, continuous, bi-directional file synchronization.",
	Run:   rootMain,
}

var rootConfiguration struct {
	help    bool
	version bool
	legal   bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := rootCommand.Flags()
	flags.BoolVarP(&rootConfiguration.help, "help", "h", false, "Show help information")
	flags.BoolVarP(&rootConfiguration.version, "version", "V", false, "Show version information")
	flags.BoolVarP(&rootConfiguration.legal, "legal", "l", false, "Show legal information")

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
		pauseCommand,
		resumeCommand,
		terminateCommand,
		daemonCommand,
	)
}

func main() {
	// Check if a prompting environment is set. If so, treat this as a prompt
	// request. Prompting is sort of a special pseudo-command that's indicated
	// by the presence of environment variables, and hence it has to be handled
	// in a bit of a special manner.
	if _, ok := environment.Current[ssh.PrompterEnvironmentVariable]; ok {
		if len(os.Args) != 2 {
			cmd.Fatal(errors.New("incorrect number of prompt arguments"))
		} else if err := prompt(os.Args[1]); err != nil {
			cmd.Fatal(err)
		}
		return
	}

	// Execute the root command.
	if err := rootCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
