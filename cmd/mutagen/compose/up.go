package compose

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/pkg/compose"
)

// ensureMutagenUp ensures that the Mutagen service is running and up-to-date.
func ensureMutagenUp(topLevelFlags []string) error {
	// Set up command flags and arguments. We honor certain up command flags
	// that control output.
	var arguments []string
	arguments = append(arguments, topLevelFlags...)
	arguments = append(arguments, "up", "--detach")
	if upConfiguration.noColor {
		arguments = append(arguments, "--no-color")
	}
	if upConfiguration.quietPull {
		arguments = append(arguments, "--quiet-pull")
	}
	arguments = append(arguments, compose.MutagenServiceName)

	// Create the command.
	up, err := compose.Command(context.Background(), arguments...)
	if err != nil {
		return fmt.Errorf("unable to set up Docker Compose invocation: %w", err)
	}

	// Set up the environment. We request that the up command ignore orphaned
	// containers because we'd prefer to have those handled by the nominal up
	// command.
	up.Env = os.Environ()
	up.Env = append(up.Env, "COMPOSE_IGNORE_ORPHANS=true")

	// Forward input and output streams.
	up.Stdin = os.Stdin
	up.Stdout = os.Stdout
	up.Stderr = os.Stderr

	// Invoke the command.
	if err := up.Run(); err != nil {
		return fmt.Errorf("up command failed: %w", err)
	}

	// Success.
	return nil
}

func upMain(_ *cobra.Command, arguments []string) error {
	// Ensure that the user isn't trying to interact with the Mutagen service
	// directly.
	for _, argument := range arguments {
		if argument == compose.MutagenServiceName {
			return errors.New("the Mutagen service should not be controlled directly")
		}
	}

	// Load project metadata and defer the release of project resources.
	project, err := compose.LoadProject(
		rootConfiguration.ProjectFlags,
		rootConfiguration.DaemonConnectionFlags,
	)
	if err != nil {
		return fmt.Errorf("unable to load project: %w", err)
	}
	defer project.Dispose()

	// Compute the effective top-level flags that we'll use.
	topLevelFlags := topLevelFlags(true)
	topLevelFlags = append(topLevelFlags, project.TopLevelFlags()...)

	// Ensure that the Mutagen service is running and up-to-date.
	if err := ensureMutagenUp(topLevelFlags); err != nil {
		return fmt.Errorf("unable to bring up Mutagen service: %w", err)
	}

	// Handle Mutagen session reconciliation and initial synchronization.
	// TODO: Implement.

	// Invoke the target command.
	// TODO: Implement.

	// Success.
	return nil
}

var upCommand = &cobra.Command{
	Use:          "up",
	RunE:         composeEntryPointE(upMain),
	SilenceUsage: true,
}

var upConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// detach indicates the presence of the -d/--detach flag.
	detach bool
	// noColor indicates the presence of the --no-color flag.
	noColor bool
	// quietPull indicates the presence of the --quiet-pull flag.
	quietPull bool
	// noDeps indicates the presence of the --no-deps flag.
	noDeps bool
	// forceRecreate indicates the presence of the --force-recreate flag.
	forceRecreate bool
	// alwaysRecreateDeps indicates the presence of the --always-recreate-deps
	// flag.
	alwaysRecreateDeps bool
	// noRecreate indicates the presence of the --no-recreate flag.
	noRecreate bool
	// noBuild indicates the presence of the --no-build flag.
	noBuild bool
	// noStart indicates the presence of the --no-start flag.
	noStart bool
	// build indicates the presence of the --build flag.
	build bool
	// abortOnContainerExit indicates the presence of the
	// --abort-on-container-exit flag.
	abortOnContainerExit bool
	// attachDependencies indicates the presence of the --attach-dependencies
	// flag.
	attachDependencies bool
	// timeout stores the value of the -t/--timeout flag.
	timeout string
	// renewAnonVolumes indicates the presence of the -V/--renew-anon-volumes
	// flag.
	renewAnonVolumes bool
	// removeOrphans indicates the presence of the --remove-orphans flag.
	removeOrphans bool
	// exitCodeFrom stores the value of the --exit-code-from flag.
	exitCodeFrom string
	// scale stores the value(s) of the --scale flag(s).
	scale []string
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	upCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := upCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information.
	flags.BoolVarP(&upConfiguration.help, "help", "h", false, "")
	flags.BoolVarP(&upConfiguration.detach, "detach", "d", false, "")
	flags.BoolVar(&upConfiguration.noColor, "no-color", false, "")
	flags.BoolVar(&upConfiguration.quietPull, "quiet-pull", false, "")
	flags.BoolVar(&upConfiguration.noDeps, "no-deps", false, "")
	flags.BoolVar(&upConfiguration.forceRecreate, "force-recreate", false, "")
	flags.BoolVar(&upConfiguration.alwaysRecreateDeps, "always-recreate-deps", false, "")
	flags.BoolVar(&upConfiguration.noRecreate, "no-recreate", false, "")
	flags.BoolVar(&upConfiguration.noBuild, "no-build", false, "")
	flags.BoolVar(&upConfiguration.noStart, "no-start", false, "")
	flags.BoolVar(&upConfiguration.build, "build", false, "")
	flags.BoolVar(&upConfiguration.abortOnContainerExit, "abort-on-container-exit", false, "")
	flags.BoolVar(&upConfiguration.attachDependencies, "attach-dependencies", false, "")
	flags.StringVarP(&upConfiguration.timeout, "timeout", "t", "", "")
	flags.BoolVarP(&upConfiguration.renewAnonVolumes, "renew-anon-volumes", "V", false, "")
	flags.BoolVar(&upConfiguration.removeOrphans, "remove-orphans", false, "")
	flags.StringVar(&upConfiguration.exitCodeFrom, "exit-code-from", "", "")
	flags.StringSliceVar(&upConfiguration.scale, "scale", nil, "")
}
