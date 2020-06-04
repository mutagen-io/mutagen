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

// topLevelProjectFlagNames are the names of the top-level flags that control
// project resolution.
var topLevelProjectFlagNames = []string{
	"file",
	"project-name",
	"project-directory",
	"env-file",
}

// reconstituteFlags reconstitutes a parsed flag set, optionally filtering out
// specific flag names. It only supports the flag types needed by the compose
// command and its subcommands.
func reconstituteFlags(flags *pflag.FlagSet, exclude []string) []string {
	// Convert the exclusion list to a map.
	var excludeMap map[string]bool
	if len(exclude) > 0 {
		excludeMap = make(map[string]bool, len(exclude))
		for _, flag := range exclude {
			excludeMap[flag] = true
		}
	}

	// Perform reconstruction.
	var result []string
	flags.Visit(func(flag *pflag.Flag) {
		// Ignore excluded flags.
		if excludeMap[flag.Name] {
			return
		}

		// Handle flags based on type.
		switch flag.Value.Type() {
		case "bool":
			result = append(result, "--"+flag.Name)
		case "string":
			result = append(result, "--"+flag.Name, flag.Value.String())
		case "stringSlice":
			sliceValue, ok := flag.Value.(pflag.SliceValue)
			if !ok {
				panic("stringSlice flag did not have SliceValue type")
			}
			for _, value := range sliceValue.GetSlice() {
				result = append(result, "--"+flag.Name, value)
			}
		default:
			panic("unhandled flag type")
		}
	})

	// Done.
	return result
}

// invoke invokes Docker Compose with the specified top-level flags, command
// name, and arguments. It forwards the standard input/output/error streams and
// environment of the current process to Docker Compose. It returns any error
// that occurred while invoking Docker Compose or the result of os/exec.Cmd.Run.
func invoke(topLevelFlags []string, command string, arguments []string) error {
	// Compute the Docker Compose arguments.
	composeArguments := make([]string, 0, len(topLevelFlags)+1+len(arguments))
	composeArguments = append(composeArguments, topLevelFlags...)
	if command != "" {
		composeArguments = append(composeArguments, command)
		composeArguments = append(composeArguments, arguments...)
	}

	// Set up the Docker Compose command.
	compose, err := compose.Command(context.Background(), composeArguments...)
	if err != nil {
		return fmt.Errorf("unable to set up Docker Compose invocation: %w", err)
	}

	// Forward input and output streams.
	compose.Stdin = os.Stdin
	compose.Stdout = os.Stdout
	compose.Stderr = os.Stderr

	// TODO: Figure out if there's any signal handling that we need to set up.
	// Docker Compose has a bunch of internal signal handling at its entry
	// point, but this may not be necessary with the Go runtime in the same way
	// that it is with the Python runtime. In any case, we'll likely need to
	// forward signals.

	// Run the command and wrap any error that's not an exit error.
	if err := compose.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return fmt.Errorf("unable to invoke Docker Compose: %w", err)
		}
		return err
	}

	// Success.
	return nil
}

// invokeAndExit runs invoke with the specified parameters and terminates the
// current process with a matching exit code (or an error message and error exit
// code if the Docker Compose command failed to start).
func invokeAndExit(topLevelFlags []string, command string, arguments []string) {
	// Run invoke. If there's no error, then we can exit successfully as well.
	err := invoke(topLevelFlags, command, arguments)
	if err == nil {
		os.Exit(0)
	}

	// Otherwise attempt to extract the exit code. If the exit code is invalid,
	// then just use a standard error exit code.
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitCode := exitErr.ExitCode(); exitCode >= 1 {
			os.Exit(exitCode)
		}
		os.Exit(1)
	}

	// At this point, some other error must have occurred.
	cmd.Fatal(err)
}

// handleTopLevelFlags handles top-level Docker Compose flags. This is necessary
// to emulate Docker Compose's handling of these flags, which occurs even if a
// command is specified. If this function returns, then execution can continue
// normally.
func handleTopLevelFlags() {
	// Handle help and version flags. Help behavior always take precedence over
	// version behavior, even if the -v/--version flag is specified before the
	// -h/--help flag.
	if composeConfiguration.help {
		invokeAndExit([]string{"--help"}, "", nil)
	} else if composeConfiguration.version {
		invokeAndExit([]string{"--version"}, "", nil)
	}

	// Enforce that the --skip-hostname-check flag isn't specified. This flag
	// isn't currently supported by Mutagen's Docker transport because it isn't
	// supported by the Docker CLI.
	if composeConfiguration.skipHostnameCheck {
		cmd.Fatal(errors.New("--skip-hostname-check flag not supported by Mutagen"))
	}
}

// passthrough is a generic Cobra handler that will pass handling directly to
// Docker Compose using the command name, reconstituted top-level flags, and
// command arguments. This handler will also honor/handle top-level flags that
// result in termination. In order to use this handler, flag parsing must be
// disabled for the command.
func passthrough(command *cobra.Command, arguments []string) {
	// Handle top-level flags that might result in termination.
	handleTopLevelFlags()

	// Reconstitute top-level flags and pass control to Docker Compose.
	topLevelFlags := reconstituteFlags(composeCommand.Flags(), nil)
	invokeAndExit(topLevelFlags, command.CalledAs(), arguments)
}

// wrapper adapts an error-returning Cobra entry point to handle top-level
// Docker Compose flags and emulate Docker Compose's exit behavior. It is
// designed to be used for those handlers that perform additional logic around
// Docker Compose commands but end their operation with a call to invoke.
func wrapper(run func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(command *cobra.Command, arguments []string) error {
		// Handle top-level flags that might result in termination.
		handleTopLevelFlags()

		// Run the underlying handler.
		err := run(command, arguments)

		// If there's an exit error, then terminate the current process with the
		// same exit code.
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitCode := exitErr.ExitCode(); exitCode >= 1 {
					os.Exit(exitCode)
				}
				os.Exit(1)
			}
		}

		// Otherwise just return the error directly.
		return err
	}
}

// commandHelp is a Cobra help function that shells out to Docker Compose to
// display help information for Docker Compose commands.
func commandHelp(command *cobra.Command, _ []string) {
	if command == composeCommand {
		invokeAndExit([]string{"--help"}, "", nil)
	}
	invokeAndExit(nil, command.CalledAs(), []string{"--help"})
}

// composeMain is the entry point for the compose command.
func composeMain(_ *cobra.Command, arguments []string) error {
	// If no arguments have been specified, then just print help information,
	// but do so in a way that matches the output stream and exit code that
	// Docker Compose would use.
	if len(arguments) == 0 {
		invokeAndExit(nil, "", nil)
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
	return fmt.Errorf("unknown or unsupported command: %s", arguments[0])
}

// composeCommand is the root command of the compose command hierarchy.
var composeCommand = &cobra.Command{
	Use:              "compose",
	Short:            "Run Docker Compose with Mutagen enhancements",
	RunE:             wrapper(composeMain),
	SilenceUsage:     true,
	TraverseChildren: true,
}

// composeConfiguration stores configuration for the compose command.
var composeConfiguration struct {
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
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	composeCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := composeCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information.
	flags.BoolVarP(&composeConfiguration.help, "help", "h", false, "")
	flags.StringSliceVarP(&composeConfiguration.File, "file", "f", nil, "")
	flags.StringVarP(&composeConfiguration.ProjectName, "project-name", "p", "", "")
	flags.StringVarP(&composeConfiguration.Context, "context", "c", "", "")
	flags.BoolVar(&composeConfiguration.verbose, "verbose", false, "")
	flags.StringVar(&composeConfiguration.logLevel, "log-level", "", "")
	flags.BoolVar(&composeConfiguration.noANSI, "no-ansi", false, "")
	flags.BoolVarP(&composeConfiguration.version, "version", "v", false, "")
	flags.StringVarP(&composeConfiguration.Host, "host", "H", "", "")
	flags.BoolVar(&composeConfiguration.TLS, "tls", false, "")
	flags.StringVar(&composeConfiguration.TLSCACert, "tlscacert", "", "")
	flags.StringVar(&composeConfiguration.TLSCert, "tlscert", "", "")
	flags.StringVar(&composeConfiguration.TLSKey, "tlskey", "", "")
	flags.BoolVar(&composeConfiguration.TLSVerify, "tlsverify", false, "")
	flags.BoolVar(&composeConfiguration.skipHostnameCheck, "skip-hostname-check", false, "")
	flags.StringVar(&composeConfiguration.ProjectDirectory, "project-directory", "", "")
	flags.BoolVar(&composeConfiguration.compatibility, "compatibility", false, "")
	flags.StringVar(&composeConfiguration.EnvFile, "env-file", "", "")

	// Register commands.
	composeCommand.AddCommand(
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

// rootMain is the entry point for RootCommand. It switches control flow from
// the Mutagen command hierarchy to the Docker Compose command hierarchy.
func rootMain(_ *cobra.Command, arguments []string) {
	// Set the default argument source for the Docker Compose command hierarchy.
	composeCommand.SetArgs(arguments)

	// Execute the root command of the Docker Compose command hierarchy.
	if err := composeCommand.Execute(); err != nil {
		os.Exit(1)
	}
}

// RootCommand is an adapter command that shifts control flow from the Mutagen
// command hierarchy to the Docker Compose command hierarchy. The Docker Compose
// command hierarchy uses complex Cobra behaviors (such as TraverseChildren,
// SetInterspersed, and DisableFlagParsing) to emulate Docker Compose behavior,
// and these behaviors make it better suited to operate as a detached command
// hierarchy. TraverseChildren in particular has to be applied at the hierarchy
// root and changes certain behaviors (such as command-not-found errors), so we
// really want to isolate its effects.
var RootCommand = &cobra.Command{
	Use:                composeCommand.Use,
	Short:              composeCommand.Short,
	Run:                rootMain,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

func init() {
	// Mark the command as experimental.
	RootCommand.Short = RootCommand.Short + color.YellowString(" [Experimental]")
}
