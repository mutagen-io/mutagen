package compose

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fatih/color"

	"github.com/mutagen-io/mutagen/cmd"
)

// handleTopLevelHelp handles top-level -h/--help flags. Docker Compose will
// give these flags (and their exit behavior) priority over any additional flags
// or commands, so to match this behavior, this function needs to be called by
// all command handlers. If it returns, then the handler should proceed
// normally. Since -h/--help flags take priority over -v/--version flags (even
// when the latter is specified first), this function should be called before
// handleTopLevelVersion.
func handleTopLevelHelp() {
	if rootConfiguration.help {
		compose([]string{"--help"}, nil, nil, true)
	}
}

// handleTopLevelVersion handles top-level -v/--version flags. Docker Compose
// will give these flags (and their exit behavior) priority over any additional
// flags or commands, so to match this behavior, this function needs to be
// called by all command handlers. If it returns, then the handler should
// proceed normally. Since -h/--help flags take priority over -v/--version flags
// (even when the latter is specified first), this function should be called
// after handleTopLevelHelp.
func handleTopLevelVersion() {
	if rootConfiguration.version {
		compose([]string{"--version"}, nil, nil, true)
	}
}

func rootMain(_ *cobra.Command, arguments []string) {
	// Handle top-level help and version flags.
	handleTopLevelHelp()
	handleTopLevelVersion()

	// If no arguments have been specified, then just print help information,
	// but do so in a way that matches the output stream and exit code that
	// Docker Compose would use.
	if len(arguments) == 0 {
		compose(nil, nil, nil, true)
	}

	// Handle unknown commands. We can't precisely emulate what Docker Compose
	// does here without passing the unknown command to Docker Compose itself
	// (which we don't want to do because it could be a command that we should
	// wrap but just don't know about yet). However, we already can't exactly
	// emulate Docker Compose here because Docker Compose treats "--" as a
	// command specification (and aliases it to "__"), but our flag parsing
	// treats this as an argument terminator and ignores it. In any case, what
	// we do here should be sufficient for every conceivable case. We try to
	// match what Cobra would do, but also add information that might help users
	// understand why the command isn't yet known.
	cmd.Fatal(fmt.Errorf("unknown or unsupported command \"%s\" for \"compose\"", arguments[0]))
}

var RootCommand = &cobra.Command{
	Use:          "compose",
	Short:        "Run Docker Compose with Mutagen enhancements",
	Run:          rootMain,
	SilenceUsage: true,
}

var rootConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// files stores the value(s) of the -f/--file flag(s).
	files []string
	// projectName stores the value of the -p/--project-name flag.
	projectName string
	// verbose indicates the presence of the --verbose flag.
	verbose bool
	// logLevel stores the value of the --log-level flag.
	logLevel string
	// noANSI indicates the presence of the --no-ansi flag.
	noANSI bool
	// version indicates the presence of the -v/--version flag.
	version bool
	// host stores the value of the -H/--host flag.
	host string
	// tls indicates the presence of the --tls flag.
	tls bool
	// tlsCACert stores the value of the --tlscacert flag.
	tlsCACert string
	// tlsCert stores the value of the --tlscert flag.
	tlsCert string
	// tlsKey stores the value of the --tlskey flag.
	tlsKey string
	// tlsVerify indicates the presence of the --tlsverify flag.
	tlsVerify bool
	// skipHostnameCheck indicates the presence of the --skip-hostname-check
	// flag.
	skipHostnameCheck bool
	// projectDirectory stores the value of the --project-directory flag.
	projectDirectory string
	// compatibility indicates the presence of the --compatibility flag.
	compatibility bool
	// envFile stores the value of the --env-file flag.
	envFile string
}

func init() {
	// Mark the command as experimental.
	RootCommand.Short = RootCommand.Short + color.YellowString(" [Experimental]")

	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present and instead just redirect control to the
	// nominal entry point. We'll use the -h/--help flag that we create below to
	// determine when help functionality needs to be displayed.
	RootCommand.SetHelpFunc(rootMain)

	// Grab a handle for the command line flags.
	flags := RootCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information.
	flags.BoolVarP(&rootConfiguration.help, "help", "h", false, "")
	flags.StringSliceVarP(&rootConfiguration.files, "file", "f", nil, "")
	flags.StringVarP(&rootConfiguration.projectName, "project-name", "p", "", "")
	flags.BoolVar(&rootConfiguration.verbose, "verbose", false, "")
	flags.StringVar(&rootConfiguration.logLevel, "log-level", "", "")
	flags.BoolVar(&rootConfiguration.noANSI, "no-ansi", false, "")
	flags.BoolVarP(&rootConfiguration.version, "version", "v", false, "")
	flags.StringVarP(&rootConfiguration.host, "host", "H", "", "")
	flags.BoolVar(&rootConfiguration.tls, "tls", false, "")
	flags.StringVar(&rootConfiguration.tlsCACert, "tlscacert", "", "")
	flags.StringVar(&rootConfiguration.tlsCert, "tlscert", "", "")
	flags.StringVar(&rootConfiguration.tlsKey, "tlskey", "", "")
	flags.BoolVar(&rootConfiguration.tlsVerify, "tlsverify", false, "")
	flags.BoolVar(&rootConfiguration.skipHostnameCheck, "skip-hostname-check", false, "")
	flags.StringVar(&rootConfiguration.projectDirectory, "project-directory", "", "")
	flags.BoolVar(&rootConfiguration.compatibility, "compatibility", false, "")
	flags.StringVar(&rootConfiguration.envFile, "env-file", "", "")

	// Register commands.
	RootCommand.AddCommand(
		buildCommand,
		configCommand,
		createCommand,
		downCommand,
		eventsCommand,
		execCommand,
		helpCommand,
		imagesCommand,
		killCommand,
		logsCommand,
		pauseCommand,
		portCommand,
		psCommand,
		pullCommand,
		pushCommand,
		restartCommand,
		rmCommand,
		runCommand,
		scaleCommand,
		startCommand,
		stopCommand,
		topCommand,
		unpauseCommand,
		upCommand,
		versionCommand,
	)

	// Disable interspersed positional arguments when parsing flags. This is
	// required in order to handle unknown subcommands. With interspersed
	// arguments allowed (the default for the pflag package), arguments that are
	// neither flags nor flag values will be treated as positional arguments if
	// no matching command or alias is found, but parsing will continue past the
	// argument and likely fail when it hits an unknown flag meant for the
	// unknown command. It's possible to skip past unknown flags with Cobra, but
	// that also means unknown top-level flags will be skipped (and the rules
	// for this skipping are... iffy). However, by disabling interspersed
	// arguments, flag parsing will halt on the first non-flag/non-flag-value
	// argument and all remaining arguments (flag or otherwise) will be gathered
	// into the argument list passed to the handler, allowing us to handle
	// unknown commands more gracefully.
	flags.SetInterspersed(false)
}
