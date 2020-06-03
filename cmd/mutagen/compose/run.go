package compose

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"
)

func runPassthrough(command *cobra.Command, arguments []string) {
	cmd.Warning("The \"run\" command isn't fully supported by Mutagen.")
	fmt.Fprintln(os.Stderr, "Use \"docker-compose run\" directly to suppress this message.")
	fmt.Println()
	passthrough(command, arguments)
}

var runCommand = &cobra.Command{
	Use:                "run",
	Run:                runPassthrough,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

var runConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// detach indicates the presence of the -d/--detach flag.
	detach bool
	// name stores the value of the --name flag.
	name string
	// entrypoint stores the value of the --entrypoint flag.
	entrypoint string
	// e stores the value(s) of the -e flag(s).
	e []string
	// label stores the value(s) of the -l/--label flag(s).
	label []string
	// user stores the value of the -u/--user flag.
	user string
	// noDeps indicates the presence of the --no-deps flag.
	noDeps bool
	// rm indicates the presence of the --rm flag.
	rm bool
	// publish stores the value(s) of the -p/--publish flag(s).
	publish []string
	// servicePorts indicates the presence of the --service-ports flag.
	servicePorts bool
	// useAliases indicates the presence of the --use-aliases flag.
	useAliases bool
	// volume stores the value(s) of the -v/--volume flag(s).
	volume []string
	// T indicates the presence of the -T flag.
	t bool
	// workdir stores the value of the -w/--workdir flag.
	workdir string
}

func init() {
	// We don't set an explicit help function since we disable flag parsing for
	// this command and simply pass arguments directly through to the underlying
	// command. We still explicitly register a -h/--help flag below for shell
	// completion support.

	// Grab a handle for the command line flags.
	flags := runCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information. In
	// the case of this command, we also disable flag parsing and shell out
	// directly, so we only register these flags to support shell completion.
	flags.BoolVarP(&runConfiguration.help, "help", "h", false, "")
	flags.StringVar(&runConfiguration.name, "name", "", "")
	flags.StringVar(&runConfiguration.entrypoint, "entrypoint", "", "")
	// TODO: Figure out how to do a short-only flag for -e. See the comment on
	// the -T flag for exec to understand why this isn't currently possible.
	flags.StringSliceVarP(&runConfiguration.label, "label", "l", nil, "")
	flags.StringVarP(&runConfiguration.user, "user", "u", "", "")
	flags.BoolVar(&runConfiguration.noDeps, "no-deps", false, "")
	flags.BoolVar(&runConfiguration.rm, "rm", false, "")
	flags.StringSliceVarP(&runConfiguration.publish, "publish", "p", nil, "")
	flags.BoolVar(&runConfiguration.servicePorts, "service-ports", false, "")
	flags.BoolVar(&runConfiguration.useAliases, "use-aliases", false, "")
	flags.StringSliceVarP(&runConfiguration.volume, "volume", "v", nil, "")
	// TODO: Figure out how to do a short-only flag for -T. See the comment on
	// the -T flag for exec to understand why this isn't currently possible.
	flags.StringVarP(&runConfiguration.workdir, "workdir", "w", "", "")
}
