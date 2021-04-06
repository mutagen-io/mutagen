package compose

import (
	"github.com/spf13/cobra"
)

// configCommand is the config command.
var configCommand = &cobra.Command{
	Use:                "config",
	Run:                passthrough,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

// configConfiguration stores configuration for the config command.
var configConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// resolveImageDigests indicates the presence of the --resolve-image-digests
	// flag.
	resolveImageDigests bool
	// noInterpolate indicates the presence of the --no-interpolate flag.
	noInterpolate bool
	// quiet indicates the presence of the -q/--quiet flag.
	quiet bool
	// services indicates the presence of the --services flag.
	services bool
	// volumes indicates the presence of the --volumes flag.
	volumes bool
	// hash stores the value of the --hash flag.
	hash string
}

func init() {
	// We don't set an explicit help function since we disable flag parsing for
	// this command and simply pass arguments directly through to the underlying
	// command. We still explicitly register a -h/--help flag below for shell
	// completion support.

	// Grab a handle for the command line flags.
	flags := configCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information. In
	// the case of this command, we also disable flag parsing and shell out
	// directly, so we only register these flags to support shell completion.
	flags.BoolVarP(&configConfiguration.help, "help", "h", false, "")
	flags.BoolVar(&configConfiguration.resolveImageDigests, "resolve-image-digests", false, "")
	flags.BoolVar(&configConfiguration.noInterpolate, "no-interpolate", false, "")
	flags.BoolVarP(&configConfiguration.quiet, "quiet", "q", false, "")
	flags.BoolVar(&configConfiguration.services, "services", false, "")
	flags.BoolVar(&configConfiguration.volumes, "volumes", false, "")
	flags.StringVar(&configConfiguration.hash, "hash", "", "")
}
