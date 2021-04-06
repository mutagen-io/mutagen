package compose

import (
	"github.com/spf13/cobra"
)

// execCommand is the exec command.
var execCommand = &cobra.Command{
	Use:                "exec",
	Run:                passthrough,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

// execConfiguration stores configuration for the exec command.
var execConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// detach indicates the presence of the -d/--detach flag.
	detach bool
	// privileged indicates the presence of the --privileged flag.
	privileged bool
	// user stores the value of the -u/--user flag.
	user string
	// t indicates the presence of the -T flag.
	t bool
	// index stores the value of the --index flag.
	index string
	// env stores the value(s) of the -e/--env flag(s).
	env []string
	// workdir stores the value of the -w/--workdir flag.
	workdir string
}

func init() {
	// We don't set an explicit help function since we disable flag parsing for
	// this command and simply pass arguments directly through to the underlying
	// command. We still explicitly register a -h/--help flag below for shell
	// completion support.

	// Grab a handle for the command line flags.
	flags := execCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information. In
	// the case of this command, we also disable flag parsing and shell out
	// directly, so we only register these flags to support shell completion.
	flags.BoolVarP(&execConfiguration.help, "help", "h", false, "")
	flags.BoolVarP(&execConfiguration.detach, "detach", "d", false, "")
	flags.BoolVar(&execConfiguration.privileged, "privileged", false, "")
	flags.StringVarP(&execConfiguration.user, "user", "u", "", "")
	// TODO: Figure out how to do a short-only flag for -T. The spf13/pflag
	// package doesn't currently offer a mechanism for adding a flag with only a
	// short form (see spf13/pflag#139). The only workaround at the moment is to
	// add a long option like --T with a corresponding -T short form (and note
	// that the hack of trying to add a Go flag package flag to pflag will yield
	// the same result). Since not adding this flag here only costs us shell
	// completion, it seems best to avoid it. This is especially true since it's
	// a single character flag and the user would have to type that character to
	// disambiguate completion anyway. The risk of having --T injected into the
	// command line seems like the bigger hazard.
	flags.StringVar(&execConfiguration.index, "index", "", "")
	flags.StringSliceVarP(&execConfiguration.env, "env", "e", nil, "")
	flags.StringVarP(&execConfiguration.workdir, "workdir", "w", "", "")
}
