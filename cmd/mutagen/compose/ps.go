package compose

import (
	"github.com/spf13/cobra"
)

var psCommand = &cobra.Command{
	Use:                "ps",
	Run:                passthrough,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

var psConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// quiet indicates the presence of the -q/--quiet flag.
	quiet bool
	// services indicates the presence of the --services flag.
	services bool
	// filter stores the value of the --filter flag.
	filter string
	// all indicates the presence of the -a/--all flag.
	all bool
}

func init() {
	// We don't set an explicit help function since we disable flag parsing for
	// this command and simply pass arguments directly through to the underlying
	// command. We still explicitly register a -h/--help flag below for shell
	// completion support.

	// Grab a handle for the command line flags.
	flags := psCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information. In
	// the case of this command, we also disable flag parsing and shell out
	// directly, so we only register these flags to support shell completion.
	flags.BoolVarP(&psConfiguration.help, "help", "h", false, "")
	flags.BoolVarP(&psConfiguration.quiet, "quiet", "q", false, "")
	flags.BoolVar(&psConfiguration.services, "services", false, "")
	flags.StringVar(&psConfiguration.filter, "filter", "", "")
	flags.BoolVarP(&psConfiguration.all, "all", "a", false, "")
}
