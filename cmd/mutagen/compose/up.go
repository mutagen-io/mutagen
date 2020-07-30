package compose

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"
	"github.com/mutagen-io/mutagen/cmd/mutagen/forward"
	"github.com/mutagen-io/mutagen/cmd/mutagen/sync"

	"github.com/mutagen-io/mutagen/pkg/compose"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	"github.com/mutagen-io/mutagen/pkg/selection"
	forwardingsvc "github.com/mutagen-io/mutagen/pkg/service/forwarding"
	synchronizationsvc "github.com/mutagen-io/mutagen/pkg/service/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
)

const (
	// mutagenScalePrefix is the prefix that would be used to scale the Mutagen
	// service in a --scale flag.
	mutagenScalePrefix = compose.MutagenServiceName + "="
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

	// Set up the environment. We ignore orphaned containers because we want
	// those to be handled by the nominal up command.
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

// forwardingSessionCurrent determines whether or not an existing forwarding
// session is equivalent to the specification for its creation.
func forwardingSessionCurrent(
	session *forwarding.Session,
	specification *forwardingsvc.CreationSpecification,
) bool {
	return session.Source.Equal(specification.Source) &&
		session.Destination.Equal(specification.Destination) &&
		session.Configuration.Equal(specification.Configuration) &&
		session.ConfigurationSource.Equal(specification.ConfigurationSource) &&
		session.ConfigurationDestination.Equal(specification.ConfigurationDestination)
}

// synchronizationSessionCurrent determines whether or not an existing
// synchronization session is equivalent to the specification for its creation.
func synchronizationSessionCurrent(
	session *synchronization.Session,
	specification *synchronizationsvc.CreationSpecification,
) bool {
	return session.Alpha.Equal(specification.Alpha) &&
		session.Beta.Equal(session.Beta) &&
		session.Configuration.Equal(specification.Configuration) &&
		session.ConfigurationAlpha.Equal(specification.ConfigurationAlpha) &&
		session.ConfigurationBeta.Equal(specification.ConfigurationBeta)
}

// reconcileSessions handles Mutagen session reconciliation for the project.
func reconcileSessions(project *compose.Project) error {
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

	// Query existing forwarding sessions.
	forwardingListRequest := &forwardingsvc.ListRequest{Selection: projectSelection}
	forwardingListResponse, err := forwardingService.List(context.Background(), forwardingListRequest)
	if err != nil {
		return fmt.Errorf("forwarding session listing failed: %w", grpcutil.PeelAwayRPCErrorLayer(err))
	} else if err = forwardingListResponse.EnsureValid(); err != nil {
		return fmt.Errorf("invalid forwarding session listing response received: %w", err)
	}

	// Query existing synchronization sessions.
	synchronizationListRequest := &synchronizationsvc.ListRequest{Selection: projectSelection}
	synchronizationListResponse, err := synchronizationService.List(context.Background(), synchronizationListRequest)
	if err != nil {
		return fmt.Errorf("synchronization session listing failed: %w", grpcutil.PeelAwayRPCErrorLayer(err))
	} else if err = synchronizationListResponse.EnsureValid(); err != nil {
		return fmt.Errorf("invalid synchronization session listing response received: %w", err)
	}

	// Identify orphan forwarding sessions with no corresponding definition, as
	// well as any duplicate forwarding sessions. At the same time, construct a
	// map from session name to existing session.
	var forwardingPruneList []string
	forwardingNameToSession := make(map[string]*forwarding.Session)
	for _, state := range forwardingListResponse.SessionStates {
		if _, defined := project.Forwarding[state.Session.Name]; !defined {
			forwardingPruneList = append(forwardingPruneList, state.Session.Identifier)
		} else if _, duplicated := forwardingNameToSession[state.Session.Name]; duplicated {
			forwardingPruneList = append(forwardingPruneList, state.Session.Identifier)
		} else {
			forwardingNameToSession[state.Session.Name] = state.Session
		}
	}

	// Identify orphan synchronization sessions with no corresponding
	// definition, as well as any duplicate synchronization sessions. At the
	// same time, construct a map from session name to existing session.
	var synchronizationPruneList []string
	synchronizationNameToSession := make(map[string]*synchronization.Session)
	for _, state := range synchronizationListResponse.SessionStates {
		if _, defined := project.Synchronization[state.Session.Name]; !defined {
			synchronizationPruneList = append(synchronizationPruneList, state.Session.Identifier)
		} else if _, duplicated := synchronizationNameToSession[state.Session.Name]; duplicated {
			synchronizationPruneList = append(synchronizationPruneList, state.Session.Identifier)
		} else {
			synchronizationNameToSession[state.Session.Name] = state.Session
		}
	}

	// Identify forwarding sessions that need to be created or recreated.
	var forwardingCreateSpecifications []*forwardingsvc.CreationSpecification
	for name, specification := range project.Forwarding {
		if existing, ok := forwardingNameToSession[name]; !ok {
			forwardingCreateSpecifications = append(forwardingCreateSpecifications, specification)
		} else if !forwardingSessionCurrent(existing, specification) {
			forwardingPruneList = append(forwardingPruneList, existing.Identifier)
			forwardingCreateSpecifications = append(forwardingCreateSpecifications, specification)
		}
	}

	// Identify synchronization sessions that need to be created or recreated.
	var synchronizationCreateSpecifications []*synchronizationsvc.CreationSpecification
	for name, specification := range project.Synchronization {
		if existing, ok := synchronizationNameToSession[name]; !ok {
			synchronizationCreateSpecifications = append(synchronizationCreateSpecifications, specification)
		} else if !synchronizationSessionCurrent(existing, specification) {
			synchronizationPruneList = append(synchronizationPruneList, existing.Identifier)
			synchronizationCreateSpecifications = append(synchronizationCreateSpecifications, specification)
		}
	}

	// Prune orphaned and stale forwarding sessions.
	if len(forwardingPruneList) > 0 {
		fmt.Println("Pruning forwarding sessions")
		pruneSelection := &selection.Selection{Specifications: forwardingPruneList}
		if err := forward.TerminateWithSelection(daemonConnection, pruneSelection); err != nil {
			return fmt.Errorf("unable to prune orphaned/duplicate/stale forwarding sessions: %w", err)
		}
	}

	// Prune orphaned and stale synchronization sessions.
	if len(synchronizationPruneList) > 0 {
		fmt.Println("Pruning synchronization sessions")
		pruneSelection := &selection.Selection{Specifications: synchronizationPruneList}
		if err := sync.TerminateWithSelection(daemonConnection, pruneSelection); err != nil {
			return fmt.Errorf("unable to prune orphaned/duplicate/stale synchronization sessions: %w", err)
		}
	}

	// Ensure that all existing sessions are unpaused and connected. This is a
	// no-op for sessions that are already running and connected. We want to do
	// this in case the Mutagen service is being restarted after a system
	// shutdown or stop operation, in which case sessions may be waiting to
	// reconnect or paused, respectively.
	fmt.Println("Resuming existing forwarding sessions")
	if err := forward.ResumeWithSelection(daemonConnection, projectSelection); err != nil {
		return fmt.Errorf("forwarding resumption failed: %w", err)
	}
	fmt.Println("Resuming existing synchronization sessions")
	if err := sync.ResumeWithSelection(daemonConnection, projectSelection); err != nil {
		return fmt.Errorf("synchronization resumption failed: %w", err)
	}

	// Create forwarding sessions.
	for _, specification := range forwardingCreateSpecifications {
		fmt.Printf("Creating forwarding session \"%s\"\n", specification.Name)
		if _, err := forward.CreateWithSpecification(daemonConnection, specification); err != nil {
			return fmt.Errorf("unable to create forwarding session (%s): %w", specification.Name, err)
		}
	}

	// Create synchronization sessions.
	var newSynchronizationSessions []string
	for _, specification := range synchronizationCreateSpecifications {
		fmt.Printf("Creating synchronization session \"%s\"\n", specification.Name)
		if s, err := sync.CreateWithSpecification(daemonConnection, specification); err != nil {
			return fmt.Errorf("unable to create synchronization session (%s): %w", specification.Name, err)
		} else {
			newSynchronizationSessions = append(newSynchronizationSessions, s)
		}
	}

	// Flush newly created synchronization sessions.
	if len(newSynchronizationSessions) > 0 {
		fmt.Println("Performing initial synchronization")
		flushSelection := &selection.Selection{Specifications: newSynchronizationSessions}
		if err := sync.FlushWithSelection(daemonConnection, flushSelection, false); err != nil {
			return fmt.Errorf("unable to flush synchronization session(s): %w", err)
		}
	}

	// Success.
	return nil
}

// upMain is the entry point for the up command.
func upMain(command *cobra.Command, arguments []string) error {
	// Forbid direct control over the Mutagen service.
	for _, argument := range arguments {
		if argument == compose.MutagenServiceName {
			return errors.New("the Mutagen service should not be controlled directly")
		}
	}
	for _, scaling := range upConfiguration.scale {
		if strings.HasPrefix(scaling, mutagenScalePrefix) {
			return errors.New("the Mutagen service cannot be scaled")
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

	// Compute the effective top-level flags that we'll use. We reconstitute
	// flags from the root command, but filter project-related flags and replace
	// them with the fully resolved flags from the loaded project.
	topLevelFlags := reconstituteFlags(ComposeCommand.Flags(), topLevelProjectFlagNames)
	topLevelFlags = append(topLevelFlags, project.TopLevelFlags()...)

	// Ensure that the Mutagen service is running and up-to-date.
	if err := ensureMutagenUp(topLevelFlags); err != nil {
		return fmt.Errorf("unable to bring up Mutagen service: %w", err)
	}

	// Handle Mutagen session reconciliation.
	if err := reconcileSessions(project); err != nil {
		return fmt.Errorf("unable to reconcile Mutagen sessions: %w", err)
	}

	// If no services have been explicitly specified, then use a list of all
	// services defined for the project. We want to avoid having the Mutagen
	// service targeted by the nominal up command because we don't want the
	// flags for that command to affect the Mutagen service.
	if len(arguments) == 0 {
		if len(project.Services) == 0 {
			return errors.New("no services defined for project")
		} else {
			arguments = project.Services
		}
	}

	// Compute flags and arguments for the command itself.
	upArguments := reconstituteFlags(command.Flags(), nil)
	upArguments = append(upArguments, arguments...)

	// Perform the pass-through operation.
	return invoke(topLevelFlags, "up", upArguments)
}

// upCommand is the up command.
var upCommand = &cobra.Command{
	Use:          "up",
	RunE:         wrapper(upMain),
	SilenceUsage: true,
}

// upConfiguration stores configuration for the up command.
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
