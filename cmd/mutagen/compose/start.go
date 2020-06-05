package compose

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/pkg/compose"
)

// startMain is the entry point for the start command.
func startMain(command *cobra.Command, arguments []string) error {
	// Forbid direct control over the Mutagen service.
	for _, argument := range arguments {
		if argument == compose.MutagenServiceName {
			return errors.New("the Mutagen service should not be controlled directly")
		}
	}

	// Load project metadata and defer the release of project resources.
	project, err := compose.LoadProject(
		composeConfiguration.ProjectFlags,
		composeConfiguration.DaemonConnectionFlags,
	)
	if err != nil {
		return fmt.Errorf("unable to load project: %w", err)
	}
	defer project.Dispose()

	// We always want the Mutagen service to be started (if it isn't already),
	// so if services have been explicitly specified, then add the Mutagen
	// service to this list. If no services have been specified, then the
	// Mutagen service will be included in the operation implicitly.
	if len(arguments) > 0 {
		arguments = append(arguments, compose.MutagenServiceName)
	}

	// Compute the effective top-level flags that we'll use. We reconstitute
	// flags from the root command, but filter project-related flags and replace
	// them with the fully resolved flags from the loaded project.
	topLevelFlags := reconstituteFlags(ComposeCommand.Flags(), topLevelProjectFlagNames)
	topLevelFlags = append(topLevelFlags, project.TopLevelFlags()...)

	// Compute flags and arguments for the command itself.
	startArguments := reconstituteFlags(command.Flags(), nil)
	startArguments = append(startArguments, arguments...)

	// Perform the pass-through operation.
	if err := invoke(topLevelFlags, "start", startArguments); err != nil {
		return err
	}

	// Resume sessions.
	if err := resumeSessions(project); err != nil {
		return fmt.Errorf("unable to resume Mutagen sessions: %w", err)
	}

	// Success.
	return nil
}

// startCommand is the start command.
var startCommand = &cobra.Command{
	Use:          "start",
	RunE:         wrapper(startMain),
	SilenceUsage: true,
}

// startConfiguration stores configuration for the start command.
var startConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	startCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := startCommand.Flags()

	// Wire up start command flags.
	flags.BoolVarP(&startConfiguration.help, "help", "h", false, "")
}
