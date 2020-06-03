package compose

import (
	"github.com/spf13/cobra"
)

// pushCommand is the push command.
var pushCommand = &cobra.Command{
	Use:                "push",
	Run:                passthrough,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

// pushConfiguration stores configuration for the push command.
var pushConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// ignorePushFailures indicates the presence of the --ignore-push-failures
	// flag.
	ignorePushFailures bool
}

func init() {
	// We don't set an explicit help function since we disable flag parsing for
	// this command and simply pass arguments directly through to the underlying
	// command. We still explicitly register a -h/--help flag below for shell
	// completion support.

	// Grab a handle for the command line flags.
	flags := pushCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information. In
	// the case of this command, we also disable flag parsing and shell out
	// directly, so we only register these flags to support shell completion.
	flags.BoolVarP(&pushConfiguration.help, "help", "h", false, "")
	flags.BoolVar(&pushConfiguration.ignorePushFailures, "ignore-push-failures", false, "")
}
