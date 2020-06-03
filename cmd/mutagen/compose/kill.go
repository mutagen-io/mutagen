package compose

import (
	"github.com/spf13/cobra"
)

// killCommand is the kill command.
var killCommand = &cobra.Command{
	Use:                "kill",
	Run:                passthrough,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

// killConfiguration stores configuration for the kill command.
var killConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// signal stores the value of the -s flag.
	signal bool
}

func init() {
	// We don't set an explicit help function since we disable flag parsing for
	// this command and simply pass arguments directly through to the underlying
	// command. We still explicitly register a -h/--help flag below for shell
	// completion support.

	// Grab a handle for the command line flags.
	flags := killCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information. In
	// the case of this command, we also disable flag parsing and shell out
	// directly, so we only register these flags to support shell completion.
	flags.BoolVarP(&killConfiguration.help, "help", "h", false, "")
	// TODO: Figure out how to do a short-only flag for -s. See the comment on
	// the -T flag for exec to understand why this isn't currently possible.
}
