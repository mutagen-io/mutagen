package compose

import (
	"github.com/spf13/cobra"
)

// portCommand is the port command.
var portCommand = &cobra.Command{
	Use:                "port",
	Run:                passthrough,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

// portConfiguration stores configuration for the port command.
var portConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// protocol stores the value of the --protocol flag.
	protocol string
	// index stores the value of the --index flag.
	index string
}

func init() {
	// We don't set an explicit help function since we disable flag parsing for
	// this command and simply pass arguments directly through to the underlying
	// command. We still explicitly register a -h/--help flag below for shell
	// completion support.

	// Grab a handle for the command line flags.
	flags := portCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information. In
	// the case of this command, we also disable flag parsing and shell out
	// directly, so we only register these flags to support shell completion.
	flags.BoolVarP(&portConfiguration.help, "help", "h", false, "")
	flags.StringVar(&portConfiguration.protocol, "protocol", "", "")
	flags.StringVar(&portConfiguration.index, "index", "", "")
}
