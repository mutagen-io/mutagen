package compose

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"
	"github.com/mutagen-io/mutagen/cmd/mutagen/forward"
	"github.com/mutagen-io/mutagen/cmd/mutagen/sync"
	"github.com/mutagen-io/mutagen/pkg/compose"
	forwardingsvc "github.com/mutagen-io/mutagen/pkg/service/forwarding"
	synchronizationsvc "github.com/mutagen-io/mutagen/pkg/service/synchronization"
)

// pauseSessions handles Mutagen session pausing for the project.
func pauseSessions(project *compose.Project) error {
	// Connect to the Mutagen daemon and defer closure of the connection.
	daemonConnection, err := daemon.Connect(true, true)
	if err != nil {
		return fmt.Errorf("unable to connect to Mutagen daemon: %w", err)
	}
	defer daemonConnection.Close()

	// Create service clients.
	forwardingService := forwardingsvc.NewForwardingClient(daemonConnection)
	synchronizationService := synchronizationsvc.NewSynchronizationClient(daemonConnection)

	// Create a session selection for the project.
	projectSelection := project.SessionSelection()

	// Perform forwarding session pausing.
	if err := forward.PauseWithSelection(forwardingService, projectSelection); err != nil {
		return fmt.Errorf("forwarding pausing failed: %w", err)
	}

	// Perform synchronization session pausing.
	if err := sync.PauseWithSelection(synchronizationService, projectSelection); err != nil {
		return fmt.Errorf("synchronization pausing failed: %w", err)
	}

	// Success.
	return nil
}

func pauseMain(command *cobra.Command, arguments []string) error {
	// Forbid direct control over the Mutagen service.
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

	// If no services have been explicitly specified, then we're going to pause
	// the entire project (including the Mutagen service), so pause sessions.
	if len(arguments) == 0 {
		if err := pauseSessions(project); err != nil {
			return fmt.Errorf("unable to pause Mutagen sessions: %w", err)
		}
	}

	// Compute the effective top-level flags that we'll use. We reconstitute
	// flags from the root command, but filter project-related flags and replace
	// them with the fully resolve flags from the loaded project.
	topLevelFlags := reconstituteFlags(RootCommand.Flags(), topLevelProjectFlagNames)
	topLevelFlags = append(topLevelFlags, project.TopLevelFlags()...)

	// Compute flags and arguments for the command itself.
	pauseArguments := reconstituteFlags(command.Flags(), nil)
	pauseArguments = append(pauseArguments, arguments...)

	// Perform the pass-through operation.
	return invoke(topLevelFlags, "pause", pauseArguments)
}

var pauseCommand = &cobra.Command{
	Use:          "pause",
	RunE:         composeEntryPointE(pauseMain),
	SilenceUsage: true,
}

var pauseConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	pauseCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := pauseCommand.Flags()

	// Wire up pause command flags.
	flags.BoolVarP(&pauseConfiguration.help, "help", "h", false, "")
}
