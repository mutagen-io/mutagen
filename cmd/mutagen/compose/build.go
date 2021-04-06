package compose

import (
	"github.com/spf13/cobra"
)

// buildCommand is the build command.
var buildCommand = &cobra.Command{
	Use:                "build",
	Run:                passthrough,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

// buildConfiguration stores configuration for the build command.
var buildConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// buildArg stores the value(s) of the --build-arg flag(s).
	buildArg []string
	// compress indicates the presence of the --compress flag.
	compress bool
	// forceRm indicates the presence of the --force-rm flag.
	forceRm bool
	// memory stores the value of the -m/--memory flag.
	memory string
	// noCache indicates the presence of the --no-cache flag.
	noCache bool
	// noRm indicates the presence of the --no-rm flag.
	noRm bool
	// parallel indicates the presence of the --parallel flag.
	parallel bool
	// progress stores the value of the --progress flag.
	progress string
	// pull indicates the presence of the --pull flag.
	pull bool
	// quiet indicates the presence of the -q/--quiet flag.
	quiet bool
}

func init() {
	// We don't set an explicit help function since we disable flag parsing for
	// this command and simply pass arguments directly through to the underlying
	// command. We still explicitly register a -h/--help flag below for shell
	// completion support.

	// Grab a handle for the command line flags.
	flags := buildCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information. In
	// the case of this command, we also disable flag parsing and shell out
	// directly, so we only register these flags to support shell completion.
	flags.BoolVarP(&buildConfiguration.help, "help", "h", false, "")
	flags.StringSliceVar(&buildConfiguration.buildArg, "build-arg", nil, "")
	flags.BoolVar(&buildConfiguration.compress, "compress", false, "")
	flags.BoolVar(&buildConfiguration.forceRm, "force-rm", false, "")
	flags.StringVarP(&buildConfiguration.memory, "memory", "m", "", "")
	flags.BoolVar(&buildConfiguration.noCache, "no-cache", false, "")
	flags.BoolVar(&buildConfiguration.noRm, "no-rm", false, "")
	flags.BoolVar(&buildConfiguration.parallel, "parallel", false, "")
	flags.StringVar(&buildConfiguration.progress, "progress", "", "")
	flags.BoolVar(&buildConfiguration.pull, "pull", false, "")
	flags.BoolVarP(&buildConfiguration.quiet, "quiet", "q", false, "")
}
