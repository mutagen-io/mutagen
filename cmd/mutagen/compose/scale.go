package compose

import (
	"github.com/spf13/cobra"
)

// scaleCommand is the scale command.
var scaleCommand = &cobra.Command{
	Use:                "scale",
	Run:                passthrough,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

// scaleConfiguration stores configuration for the scale command.
var scaleConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// timeout stores the value of the -t/--timeout flag.
	timeout string
}

func init() {
	// We don't set an explicit help function since we disable flag parsing for
	// this command and simply pass arguments directly through to the underlying
	// command. We still explicitly register a -h/--help flag below for shell
	// completion support.

	// Grab a handle for the command line flags.
	flags := scaleCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information. In
	// the case of this command, we also disable flag parsing and shell out
	// directly, so we only register these flags to support shell completion.
	flags.BoolVarP(&scaleConfiguration.help, "help", "h", false, "")
	flags.StringVarP(&scaleConfiguration.timeout, "timeout", "t", "", "")
}
