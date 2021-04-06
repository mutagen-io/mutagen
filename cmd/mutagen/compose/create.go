package compose

import (
	"github.com/spf13/cobra"
)

// createCommand is the create command.
var createCommand = &cobra.Command{
	Use:                "create",
	Run:                passthrough,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

// createConfiguration stores configuration for the create command.
var createConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// forceRecreate indicates the presence of the --force-recreate flag.
	forceRecreate bool
	// noRecreate indicates the presence of the --no-recreate flag.
	noRecreate bool
	// noBuild indicates the presence of the --no-build flag.
	noBuild bool
	// build indicates the presence of the --build flag.
	build bool
}

func init() {
	// We don't set an explicit help function since we disable flag parsing for
	// this command and simply pass arguments directly through to the underlying
	// command. We still explicitly register a -h/--help flag below for shell
	// completion support.

	// Grab a handle for the command line flags.
	flags := createCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information. In
	// the case of this command, we also disable flag parsing and shell out
	// directly, so we only register these flags to support shell completion.
	flags.BoolVarP(&createConfiguration.help, "help", "h", false, "")
	flags.BoolVar(&createConfiguration.help, "force-recreate", false, "")
	flags.BoolVar(&createConfiguration.noRecreate, "no-recreate", false, "")
	flags.BoolVar(&createConfiguration.noBuild, "no-build", false, "")
	flags.BoolVar(&createConfiguration.build, "build", false, "")
}
