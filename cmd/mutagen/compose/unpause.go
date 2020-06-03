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

// resumeSessions handles Mutagen session resuming for the project.
func resumeSessions(project *compose.Project) error {
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

	// Perform forwarding session resumption.
	fmt.Println("Resuming forwarding sessions")
	if err := forward.ResumeWithSelection(forwardingService, projectSelection); err != nil {
		return fmt.Errorf("forwarding resumption failed: %w", err)
	}

	// Perform synchronization session resumption.
	fmt.Println("Resuming synchronization sessions")
	if err := sync.ResumeWithSelection(synchronizationService, projectSelection); err != nil {
		return fmt.Errorf("synchronization resumption failed: %w", err)
	}

	// Success.
	return nil
}

func unpauseMain(command *cobra.Command, arguments []string) error {
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

	// We always want the Mutagen service to be unpaused (if it isn't already),
	// so if services have been explicitly specified, then add the Mutagen
	// service to this list. If no services have been specified, then the
	// Mutagen service will be included in the operation implicitly.
	if len(arguments) > 0 {
		arguments = append(arguments, compose.MutagenServiceName)
	}

	// Compute the effective top-level flags that we'll use. We reconstitute
	// flags from the root command, but filter project-related flags and replace
	// them with the fully resolved flags from the loaded project.
	topLevelFlags := reconstituteFlags(composeCommand.Flags(), topLevelProjectFlagNames)
	topLevelFlags = append(topLevelFlags, project.TopLevelFlags()...)

	// Compute flags and arguments for the command itself.
	unpauseArguments := reconstituteFlags(command.Flags(), nil)
	unpauseArguments = append(unpauseArguments, arguments...)

	// Perform the pass-through operation.
	if err := invoke(topLevelFlags, "unpause", unpauseArguments); err != nil {
		return err
	}

	// Resume sessions.
	if err := resumeSessions(project); err != nil {
		return fmt.Errorf("unable to resume Mutagen sessions: %w", err)
	}

	// Success.
	return nil
}

var unpauseCommand = &cobra.Command{
	Use:          "unpause",
	RunE:         wrapper(unpauseMain),
	SilenceUsage: true,
}

var unpauseConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	unpauseCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := unpauseCommand.Flags()

	// Wire up unpause command flags.
	flags.BoolVarP(&unpauseConfiguration.help, "help", "h", false, "")
}
