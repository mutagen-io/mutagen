package compose

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"
	"github.com/mutagen-io/mutagen/cmd/mutagen/forward"
	"github.com/mutagen-io/mutagen/cmd/mutagen/sync"

	"github.com/mutagen-io/mutagen/pkg/compose"
)

// listSessions handles Mutagen session listing for the project.
func listSessions(project *compose.Project) error {
	// Connect to the Mutagen daemon and defer closure of the connection.
	daemonConnection, err := daemon.Connect(true, true)
	if err != nil {
		return fmt.Errorf("unable to connect to Mutagen daemon: %w", err)
	}
	defer daemonConnection.Close()

	// Create a session selection for the project.
	projectSelection := project.SessionSelection()

	// Perform forwarding session listing.
	fmt.Println("\nForwarding sessions")
	if err := forward.ListWithSelection(daemonConnection, projectSelection, false); err != nil {
		return fmt.Errorf("forwarding listing failed: %w", err)
	}

	// Perform synchronization session listing.
	fmt.Println("\nSynchronization sessions")
	if err := sync.ListWithSelection(daemonConnection, projectSelection, false); err != nil {
		return fmt.Errorf("synchronization listing failed: %w", err)
	}

	// Success.
	return nil
}

// psMain is the entry point for the ps command.
func psMain(command *cobra.Command, arguments []string) error {
	// Load project metadata and defer the release of project resources.
	project, err := compose.LoadProject(
		composeConfiguration.ProjectFlags,
		composeConfiguration.DaemonConnectionFlags,
	)
	if err != nil {
		return fmt.Errorf("unable to load project: %w", err)
	}
	defer project.Dispose()

	// Compute the effective top-level flags that we'll use. We reconstitute
	// flags from the root command, but filter project-related flags and replace
	// them with the fully resolved flags from the loaded project.
	topLevelFlags := reconstituteFlags(composeCommand.Flags(), topLevelProjectFlagNames)
	topLevelFlags = append(topLevelFlags, project.TopLevelFlags()...)

	// Compute flags and arguments for the command itself.
	psArguments := reconstituteFlags(command.Flags(), nil)
	psArguments = append(psArguments, arguments...)

	// Perform the pass-through operation.
	if err := invoke(topLevelFlags, "ps", psArguments); err != nil {
		return err
	}

	// If the operation completed successfully and flags/services were
	// specified, then perform a Mutagen session listing.
	if len(psArguments) == 0 {
		if err := listSessions(project); err != nil {
			return fmt.Errorf("unable to list Mutagen sessions: %w", err)
		}
	}

	// Success.
	return nil
}

// psCommand is the ps command.
var psCommand = &cobra.Command{
	Use:          "ps",
	RunE:         wrapper(psMain),
	SilenceUsage: true,
}

// psConfiguration stores configuration for the ps command.
var psConfiguration struct {
	// help indicates the presence of the -h/--help flag.
	help bool
	// quiet indicates the presence of the -q/--quiet flag.
	quiet bool
	// services indicates the presence of the --services flag.
	services bool
	// filter stores the value of the --filter flag.
	filter string
	// all indicates the presence of the -a/--all flag.
	all bool
}

func init() {
	// We don't set an explicit help function since we disable flag parsing for
	// this command and simply pass arguments directly through to the underlying
	// command. We still explicitly register a -h/--help flag below for shell
	// completion support.

	// Grab a handle for the command line flags.
	flags := psCommand.Flags()

	// Wire up flags. We don't bother specifying usage information since we'll
	// shell out to Docker Compose if we need to display help information. In
	// the case of this command, we also disable flag parsing and shell out
	// directly, so we only register these flags to support shell completion.
	flags.BoolVarP(&psConfiguration.help, "help", "h", false, "")
	flags.BoolVarP(&psConfiguration.quiet, "quiet", "q", false, "")
	flags.BoolVar(&psConfiguration.services, "services", false, "")
	flags.StringVar(&psConfiguration.filter, "filter", "", "")
	flags.BoolVarP(&psConfiguration.all, "all", "a", false, "")
}
