package compose

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/fatih/color"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/pkg/compose"
	"github.com/mutagen-io/mutagen/pkg/docker"
)

// invoke invokes Docker Compose with the specified top-level flags, command
// name, and arguments. It forwards standard input/output/error streams to the
// child process and terminates the current process with the same exit code as
// the child process. If an error occurs while trying to invoke Docker Compose,
// then this function will print an error message and terminate the current
// process with an error exit code. If command is an empty string, then no
// command is specified to Docker Compose and arguments are ignored (though
// top-level flags are still included in the Docker Compose invocation). Upon
// successful invocation, this function will terminate the current process with
// an exit code of 0 if exitOnSuccess is true, otherwise it will return control
// to the caller.
func invoke(topLevelFlags []string, command string, arguments []string, exitOnSuccess bool) {
	// Compute the Docker Compose arguments.
	composeArguments := make([]string, 0, len(topLevelFlags)+1+len(arguments))
	composeArguments = append(composeArguments, topLevelFlags...)
	if command != "" {
		composeArguments = append(composeArguments, command)
		composeArguments = append(composeArguments, arguments...)
	}

	// Set up the Docker Compose commmand.
	compose, err := compose.Command(context.Background(), composeArguments...)
	if err != nil {
		cmd.Fatal(fmt.Errorf("unable to set up Docker Compose invocation: %w", err))
	}

	// Setup input and output streams.
	compose.Stdin = os.Stdin
	compose.Stdout = os.Stdout
	compose.Stderr = os.Stderr

	// TODO: Figure out if there's any signal handling that we need to set up.
	// Docker Compose has a bunch of internal signal handling at its entry
	// point, but this may not be necessary with the Go runtime in the same way
	// that it is with the Python runtime. In any case, we'll likely need to
	// forward signals.

	// Run the command.
	if err := compose.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitCode := exitErr.ExitCode(); exitCode < 1 {
				os.Exit(1)
			} else {
				os.Exit(exitCode)
			}
		} else {
			cmd.Fatal(fmt.Errorf("unable to invoke Docker Compose: %w", err))
		}
	}

	// Success.
	if exitOnSuccess {
		os.Exit(0)
	}
}

// topLevelFlags reconstitutes parsed top-level Docker Compose flags. If
// excludeProjectFlags is true, then -f/--file, -p/--project-name,
// --project-directory, and --env-file flags will be excluded.
func topLevelFlags(excludeProjectFlags bool) (flags []string) {
	RootCommand.Flags().Visit(func(flag *pflag.Flag) {
		// Check for excluded flags.
		if excludeProjectFlags {
			switch flag.Name {
			case "file":
				return
			case "project-name":
				return
			case "project-directory":
				return
			case "env-file":
				return
			}
		}

		// Handle flags based on type.
		switch flag.Value.Type() {
		case "bool":
			flags = append(flags, "--"+flag.Name)
		case "string":
			flags = append(flags, "--"+flag.Name, flag.Value.String())
		case "stringSlice":
			sliceValue, ok := flag.Value.(pflag.SliceValue)
			if !ok {
				panic("stringSlice flag did not have SliceValue type")
			}
			for _, value := range sliceValue.GetSlice() {
				flags = append(flags, "--"+flag.Name, value)
			}
		default:
			panic("unhandled flag type")
		}
	})
	return
}

// handleTopLevelFlags handles top-level Docker Compose flags. This is necessary
// to emulate Docker Compose's handling of these flags, which occurs even if a
// command is specified. If this function returns, then execution can continue
// normally.
func handleTopLevelFlags() {
	// Handle help and version flags. Help behavior always take precedence over
	// version behavior, even if the -v/--version flag is specified before the
	// -h/--help flag.
	if rootConfiguration.help {
		invoke([]string{"--help"}, "", nil, true)
	} else if rootConfiguration.version {
		invoke([]string{"--version"}, "", nil, true)
	}

	// Enforce that the --skip-hostname-check flag isn't specified. This flag
	// isn't currently supported by Mutagen's Docker transport because it isn't
	// supported by the Docker CLI.
	if rootConfiguration.skipHostnameCheck {
		cmd.Fatal(errors.New("--skip-hostname-check flag not supported by Mutagen"))
	}
}

// composeEntryPoint adapts a standard Cobra entry point to handle top-level
// Docker Compose flags.
func composeEntryPoint(run func(*cobra.Command, []string)) func(*cobra.Command, []string) {
	return func(command *cobra.Command, arguments []string) {
		handleTopLevelFlags()
		run(command, arguments)
	}
}

// composeEntryPointE adapts an error-returning Cobra entry point to handle
// top-level Docker Compose flags.
func composeEntryPointE(run func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(command *cobra.Command, arguments []string) error {
		handleTopLevelFlags()
		return run(command, arguments)
	}
}

// passthrough is a generic Cobra handler that will pass handling directly to
// Docker Compose using the command name, reconstituted top-level flags, and
// command arguments. In order to use this handler, flag parsing must be
// disabled for the command.
func passthrough(command *cobra.Command, arguments []string) {
	invoke(topLevelFlags(false), command.CalledAs(), arguments, true)
}

// commandHelp is a Cobra help function that shells out to Docker Compose to
// display help information for Docker Compose commands.
func commandHelp(command *cobra.Command, _ []string) {
	if command == RootCommand {
		invoke([]string{"--help"}, "", nil, true)
	}
	invoke(nil, command.CalledAs(), []string{"--help"}, true)
}

func rootMain(_ *cobra.Command, arguments []string) {
	// If no arguments have been specified, then just print help information,
	// but do so in a way that matches the output stream and exit code that
	// Docker Compose would use.
	if len(arguments) == 0 {
		invoke(nil, "", nil, true)
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
	Run:          composeEntryPoint(rootMain),
	SilenceUsage: true,
}

var rootConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// ProjectFlags are the flags that control the Docker Compose project.
	compose.ProjectFlags
	// DaemonConnectionFlags are the flags that control the Docker daemon
	// connection.
	docker.DaemonConnectionFlags
	// verbose indicates the presence of the --verbose flag.
	verbose bool
	// logLevel stores the value of the --log-level flag.
	logLevel string
	// noANSI indicates the presence of the --no-ansi flag.
	noANSI bool
	// version indicates the presence of the -v/--version flag.
	version bool
	// skipHostnameCheck indicates the presence of the --skip-hostname-check
	// flag.
	skipHostnameCheck bool
	// compatibility indicates the presence of the --compatibility flag.
	compatibility bool
}

func init() {
	// Mark the command as experimental.
	RootCommand.Short = RootCommand.Short + color.YellowString(" [Experimental]")

	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	RootCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := RootCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information.
	flags.BoolVarP(&rootConfiguration.help, "help", "h", false, "")
	flags.StringSliceVarP(&rootConfiguration.File, "file", "f", nil, "")
	flags.StringVarP(&rootConfiguration.ProjectName, "project-name", "p", "", "")
	flags.StringVarP(&rootConfiguration.Context, "context", "c", "", "")
	flags.BoolVar(&rootConfiguration.verbose, "verbose", false, "")
	flags.StringVar(&rootConfiguration.logLevel, "log-level", "", "")
	flags.BoolVar(&rootConfiguration.noANSI, "no-ansi", false, "")
	flags.BoolVarP(&rootConfiguration.version, "version", "v", false, "")
	flags.StringVarP(&rootConfiguration.Host, "host", "H", "", "")
	flags.BoolVar(&rootConfiguration.TLS, "tls", false, "")
	flags.StringVar(&rootConfiguration.TLSCACert, "tlscacert", "", "")
	flags.StringVar(&rootConfiguration.TLSCert, "tlscert", "", "")
	flags.StringVar(&rootConfiguration.TLSKey, "tlskey", "", "")
	flags.BoolVar(&rootConfiguration.TLSVerify, "tlsverify", false, "")
	flags.BoolVar(&rootConfiguration.skipHostnameCheck, "skip-hostname-check", false, "")
	flags.StringVar(&rootConfiguration.ProjectDirectory, "project-directory", "", "")
	flags.BoolVar(&rootConfiguration.compatibility, "compatibility", false, "")
	flags.StringVar(&rootConfiguration.EnvFile, "env-file", "", "")

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
