package compose

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"
	"github.com/mutagen-io/mutagen/cmd/mutagen/forward"
	"github.com/mutagen-io/mutagen/cmd/mutagen/sync"

	"github.com/mutagen-io/mutagen/pkg/compose"
)

// terminateSessions handles Mutagen session termination for the project.
func terminateSessions(project *compose.Project) error {
	// Connect to the Mutagen daemon and defer closure of the connection.
	daemonConnection, err := daemon.Connect(true, true)
	if err != nil {
		return fmt.Errorf("unable to connect to Mutagen daemon: %w", err)
	}
	defer daemonConnection.Close()

	// Create a session selection for the project.
	projectSelection := project.SessionSelection()

	// Perform forwarding session termination.
	fmt.Println("Terminating forwarding sessions")
	if err := forward.TerminateWithSelection(daemonConnection, projectSelection); err != nil {
		return fmt.Errorf("forwarding termination failed: %w", err)
	}

	// Perform synchronization session termination.
	fmt.Println("Terminating synchronization sessions")
	if err := sync.TerminateWithSelection(daemonConnection, projectSelection); err != nil {
		return fmt.Errorf("synchronization termination failed: %w", err)
	}

	// Success.
	return nil
}

// downMain is the entry point for the down command.
func downMain(command *cobra.Command, arguments []string) error {
	// Forbid direct control over the Mutagen service.
	for _, argument := range arguments {
		if argument == compose.MutagenServiceName {
			return errors.New("the Mutagen service should not be controlled directly")
		}
	}

	// Load project metadata and defer the release of project resources. We have
	// to do this even if service names have been explicitly specified (in which
	// case we don't shut down Mutagen sessions or the Mutagen service) because
	// down is one of two commands (the other being up) where orphan containers
	// are identified by Docker Compose, and we don't want the Mutagen service
	// to be identified as an orphan. We also don't want to disable orphan
	// detection, since it is a useful feature of this command.
	project, err := compose.LoadProject(
		composeConfiguration.ProjectFlags,
		composeConfiguration.DaemonConnectionFlags,
	)
	if err != nil {
		return fmt.Errorf("unable to load project: %w", err)
	}
	defer project.Dispose()

	// If no services have been explicitly specified, then we're going to tear
	// down the entire project. In that case we need to terminate sessions.
	if len(arguments) == 0 {
		if err := terminateSessions(project); err != nil {
			return fmt.Errorf("unable to terminate Mutagen sessions: %w", err)
		}
	}

	// Compute the effective top-level flags that we'll use. We reconstitute
	// flags from the root command, but filter project-related flags and replace
	// them with the fully resolved flags from the loaded project.
	topLevelFlags := reconstituteFlags(composeCommand.Flags(), topLevelProjectFlagNames)
	topLevelFlags = append(topLevelFlags, project.TopLevelFlags()...)

	// Compute flags and arguments for the command itself.
	downArguments := reconstituteFlags(command.Flags(), nil)
	downArguments = append(downArguments, arguments...)

	// Perform the pass-through operation.
	return invoke(topLevelFlags, "down", downArguments)
}

// downCommand is the down command.
var downCommand = &cobra.Command{
	Use:          "down",
	RunE:         wrapper(downMain),
	SilenceUsage: true,
}

// downConfiguration stores configuration for the down command.
var downConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// rmi stores the value of the --rmi flag.
	rmi string
	// volumes indicates the presence of the -v/--volumes flag.
	volumes bool
	// removeOrphans indicates the presence of the --remove-orphans flag.
	removeOrphans bool
	// timeout stores the value of the -t/--timeout flag.
	timeout string
}

func init() {
	// Avoid Cobra's built-in help functionality that's triggered when the
	// -h/--help flag is present. We still explicitly register a -h/--help flag
	// below for shell completion support.
	downCommand.SetHelpFunc(commandHelp)

	// Grab a handle for the command line flags.
	flags := downCommand.Flags()

	// Wire up down command flags.
	flags.BoolVarP(&downConfiguration.help, "help", "h", false, "")
	flags.StringVar(&downConfiguration.rmi, "rmi", "", "")
	flags.BoolVarP(&downConfiguration.volumes, "volumes", "v", false, "")
	flags.BoolVar(&downConfiguration.removeOrphans, "remove-orphans", false, "")
	flags.StringVarP(&downConfiguration.timeout, "timeout", "t", "", "")
}
