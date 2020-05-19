package compose

import (
	"github.com/spf13/cobra"
)

func helpMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// Handle command invocation.
	arguments = append([]string{"help"}, arguments...)
	compose(arguments, nil, nil, true)
}

var helpCommand = &cobra.Command{
	Use:                "help",
	Run:                helpMain,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

var helpConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := helpCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information. In
	// the case of this command, we also disable flag parsing and shell out
	// directly, so we only register these flags to support shell completion.
	flags.BoolVarP(&helpConfiguration.help, "help", "h", false, "")
}
