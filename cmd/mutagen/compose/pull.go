package compose

import (
	"github.com/spf13/cobra"
)

// pullCommand is the pull command.
var pullCommand = &cobra.Command{
	Use:                "pull",
	Run:                passthrough,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

// pullConfiguration stores configuration for the pull command.
var pullConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// ignorePullFailures indicates the presence of the --ignore-pull-failures
	// flag.
	ignorePullFailures bool
	// parallel indicates the presence of the --parallel flag.
	parallel bool
	// noParallel indicates the presence of the --no-parallel flag.
	noParallel bool
	// quiet indicates the presence of the -q/--quiet flag.
	quiet bool
	// includeDeps indicates the presence of the --include-deps flag.
	includeDeps bool
}

func init() {
	// We don't set an explicit help function since we disable flag parsing for
	// this command and simply pass arguments directly through to the underlying
	// command. We still explicitly register a -h/--help flag below for shell
	// completion support.

	// Grab a handle for the command line flags.
	flags := pullCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information. In
	// the case of this command, we also disable flag parsing and shell out
	// directly, so we only register these flags to support shell completion.
	flags.BoolVarP(&pullConfiguration.help, "help", "h", false, "")
	flags.BoolVar(&pullConfiguration.ignorePullFailures, "ignore-pull-failures", false, "")
	flags.BoolVar(&pullConfiguration.parallel, "parallel", false, "")
	flags.BoolVar(&pullConfiguration.noParallel, "no-parallel", false, "")
	flags.BoolVarP(&pullConfiguration.quiet, "quiet", "q", false, "")
	flags.BoolVar(&pullConfiguration.includeDeps, "include-deps", false, "")
}
