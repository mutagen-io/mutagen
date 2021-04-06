package compose

import (
	"github.com/spf13/cobra"
)

// logsCommand is the logs command.
var logsCommand = &cobra.Command{
	Use:                "logs",
	Run:                passthrough,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

// logsConfiguration stores configuration for the logs command.
var logsConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// noColor indicates the presence of the --no-color flag.
	noColor bool
	// follow indicates the presence of the -f/--follow flag.
	follow bool
	// timestamps indicates the presence of the -t/--timestamps flag.
	timestamps bool
	// tail stores the value of the --tail flag.
	tail string
}

func init() {
	// We don't set an explicit help function since we disable flag parsing for
	// this command and simply pass arguments directly through to the underlying
	// command. We still explicitly register a -h/--help flag below for shell
	// completion support.

	// Grab a handle for the command line flags.
	flags := logsCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information. In
	// the case of this command, we also disable flag parsing and shell out
	// directly, so we only register these flags to support shell completion.
	flags.BoolVarP(&logsConfiguration.help, "help", "h", false, "")
	flags.BoolVar(&logsConfiguration.noColor, "no-color", false, "")
	flags.BoolVarP(&logsConfiguration.follow, "follow", "f", false, "")
	flags.BoolVarP(&logsConfiguration.timestamps, "timestamps", "t", false, "")
	flags.StringVar(&logsConfiguration.tail, "tail", "", "")
}
