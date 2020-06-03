package compose

import (
	"github.com/spf13/cobra"
)

// rmCommand is the rm command.
var rmCommand = &cobra.Command{
	Use:                "rm",
	Run:                passthrough,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

// rmConfiguration stores configuration for the rm command.
var rmConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// force indicates the presence of the -f/--force flag.
	force bool
	// stop indicates the presence of the -s/--stop flag.
	stop bool
	// v indicates the presence of the -v flag.
	v bool
	// all indicates the presence of the -a/--all flag.
	all bool
}

func init() {
	// We don't set an explicit help function since we disable flag parsing for
	// this command and simply pass arguments directly through to the underlying
	// command. We still explicitly register a -h/--help flag below for shell
	// completion support.

	// Grab a handle for the command line flags.
	flags := rmCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information. In
	// the case of this command, we also disable flag parsing and shell out
	// directly, so we only register these flags to support shell completion.
	flags.BoolVarP(&rmConfiguration.help, "help", "h", false, "")
	flags.BoolVarP(&rmConfiguration.force, "force", "f", false, "")
	flags.BoolVarP(&rmConfiguration.stop, "stop", "s", false, "")
	// TODO: Figure out how to do a short-only flag for -v. See the comment on
	// the -T flag for exec to understand why this isn't currently possible.
	flags.BoolVarP(&rmConfiguration.all, "all", "a", false, "")
}
