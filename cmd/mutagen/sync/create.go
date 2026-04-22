package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"google.golang.org/grpc"

	"github.com/dustin/go-humanize"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"

	"github.com/mutagen-io/mutagen/pkg/configuration/global"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/filesystem/behavior"
	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	"github.com/mutagen-io/mutagen/pkg/selection"
	promptingsvc "github.com/mutagen-io/mutagen/pkg/service/prompting"
	synchronizationsvc "github.com/mutagen-io/mutagen/pkg/service/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/compression"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore"
	"github.com/mutagen-io/mutagen/pkg/synchronization/hashing"
	"github.com/mutagen-io/mutagen/pkg/url"
)

// loadAndValidateGlobalSynchronizationConfiguration loads a YAML-based global
// configuration, extracts the synchronization component, converts it to a
// Protocol Buffers session configuration, and validates it.
func loadAndValidateGlobalSynchronizationConfiguration(path string) (*synchronization.Configuration, error) {
	// Load the YAML configuration.
	yamlConfiguration, err := global.LoadConfiguration(path)
	if err != nil {
		return nil, err
	}

	// Convert the YAML configuration to a Protocol Buffers representation and
	// validate it.
	configuration := yamlConfiguration.Synchronization.Defaults.ToInternal()
	if err := configuration.EnsureValid(false); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Success.
	return configuration, nil
}

// CreateWithSpecification is an orchestration convenience method that performs
// a create operation using the provided daemon connection and session
// specification.
func CreateWithSpecification(
	daemonConnection *grpc.ClientConn,
	specification *synchronizationsvc.CreationSpecification,
) (string, error) {
	// Initiate command line prompting.
	statusLinePrinter := &cmd.StatusLinePrinter{}
	promptingCtx, promptingCancel := context.WithCancel(context.Background())
	prompter, promptingErrors, err := promptingsvc.Host(
		promptingCtx, promptingsvc.NewPromptingClient(daemonConnection),
		&cmd.StatusLinePrompter{Printer: statusLinePrinter}, true,
	)
	if err != nil {
		promptingCancel()
		return "", fmt.Errorf("unable to initiate prompting: %w", err)
	}

	// Perform the create operation, cancel prompting, and handle errors.
	synchronizationService := synchronizationsvc.NewSynchronizationClient(daemonConnection)
	request := &synchronizationsvc.CreateRequest{
		Prompter:      prompter,
		Specification: specification,
	}
	response, err := synchronizationService.Create(context.Background(), request)
	promptingCancel()
	<-promptingErrors
	if err != nil {
		statusLinePrinter.BreakIfPopulated()
		return "", grpcutil.PeelAwayRPCErrorLayer(err)
	} else if err = response.EnsureValid(); err != nil {
		statusLinePrinter.BreakIfPopulated()
		return "", fmt.Errorf("invalid create response received: %w", err)
	}

	// Success.
	statusLinePrinter.Clear()
	return response.Session, nil
}

// createMain is the entry point for the create command.
func createMain(_ *cobra.Command, arguments []string) error {
	// Validate, extract, and parse URLs.
	if len(arguments) != 2 {
		return errors.New("invalid number of endpoint URLs provided")
	}
	alpha, err := url.Parse(arguments[0], url.Kind_Synchronization, true)
	if err != nil {
		return fmt.Errorf("unable to parse alpha URL: %w", err)
	}
	beta, err := url.Parse(arguments[1], url.Kind_Synchronization, false)
	if err != nil {
		return fmt.Errorf("unable to parse beta URL: %w", err)
	}

	if alpha.Protocol == url.Protocol_SSH {
		sshConfigPath := os.Getenv("MUTAGEN_SSH_CONFIG_ALPHA")
		if sshConfigPath == "" {
			sshConfigPath = os.Getenv("MUTAGEN_SSH_CONFIG")
		}
		if sshConfigPath != "" {
			if alpha.Parameters == nil {
				alpha.Parameters = make(map[string]string)
			}
			alpha.Parameters["ssh-config-path"] = sshConfigPath
		}
	}
	if beta.Protocol == url.Protocol_SSH {
		sshConfigPath := os.Getenv("MUTAGEN_SSH_CONFIG_BETA")
		if sshConfigPath == "" {
			sshConfigPath = os.Getenv("MUTAGEN_SSH_CONFIG")
		}
		if sshConfigPath != "" {
			if beta.Parameters == nil {
				beta.Parameters = make(map[string]string)
			}
			beta.Parameters["ssh-config-path"] = sshConfigPath
		}
	}

	// Validate the name.
	if err := selection.EnsureNameValid(createConfiguration.name); err != nil {
		return fmt.Errorf("invalid session name: %w", err)
	}

	// Parse, validate, and record labels.
	var labels map[string]string
	if len(createConfiguration.labels) > 0 {
		labels = make(map[string]string, len(createConfiguration.labels))
	}
	for _, label := range createConfiguration.labels {
		components := strings.SplitN(label, "=", 2)
		var key, value string
		key = components[0]
		if len(components) == 2 {
			value = components[1]
		}
		if err := selection.EnsureLabelKeyValid(key); err != nil {
			return fmt.Errorf("invalid label key: %w", err)
		} else if err := selection.EnsureLabelValueValid(value); err != nil {
			return fmt.Errorf("invalid label value: %w", err)
		}
		labels[key] = value
	}

	// Create a default session configuration that will form the basis of our
	// cumulative configuration.
	configuration := &synchronization.Configuration{}

	// Unless disabled, attempt to load configuration from the global
	// configuration file and merge it into our cumulative configuration.
	if !createConfiguration.noGlobalConfiguration {
		// Compute the path to the global configuration file.
		globalConfigurationPath, err := global.ConfigurationPath()
		if err != nil {
			return fmt.Errorf("unable to compute path to global configuration file: %w", err)
		}

		// Attempt to load the file. We allow it to not exist.
		globalConfiguration, err := loadAndValidateGlobalSynchronizationConfiguration(globalConfigurationPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("unable to load global configuration: %w", err)
			}
		} else {
			configuration = synchronization.MergeConfigurations(configuration, globalConfiguration)
		}
	}

	// If additional default configuration files have been specified, then load
	// them and merge them into the cumulative configuration.
	for _, configurationFile := range createConfiguration.configurationFiles {
		if c, err := loadAndValidateGlobalSynchronizationConfiguration(configurationFile); err != nil {
			return fmt.Errorf("unable to load configuration file (%s): %w", configurationFile, err)
		} else {
			configuration = synchronization.MergeConfigurations(configuration, c)
		}
	}

	// Validate and convert the synchronization mode specification.
	var synchronizationMode core.SynchronizationMode
	if createConfiguration.synchronizationMode != "" {
		if err := synchronizationMode.UnmarshalText([]byte(createConfiguration.synchronizationMode)); err != nil {
			return fmt.Errorf("unable to parse synchronization mode: %w", err)
		}
	}

	// Validate and convert the hashing algorithm specification.
	var hashingAlgorithm hashing.Algorithm
	if createConfiguration.hash != "" {
		if err := hashingAlgorithm.UnmarshalText([]byte(createConfiguration.hash)); err != nil {
			return fmt.Errorf("unable to parse hashing algorithm: %w", err)
		}
	}

	// There's no need to validate the maximum entry count - any uint64 value is
	// valid.

	// Validate and convert the maximum staging file size.
	var maximumStagingFileSize uint64
	if createConfiguration.maximumStagingFileSize != "" {
		if s, err := humanize.ParseBytes(createConfiguration.maximumStagingFileSize); err != nil {
			return fmt.Errorf("unable to parse maximum staging file size: %w", err)
		} else {
			maximumStagingFileSize = s
		}
	}

	// Validate and convert probe mode specifications.
	var probeMode, probeModeAlpha, probeModeBeta behavior.ProbeMode
	if createConfiguration.probeMode != "" {
		if err := probeMode.UnmarshalText([]byte(createConfiguration.probeMode)); err != nil {
			return fmt.Errorf("unable to parse probe mode: %w", err)
		}
	}
	if createConfiguration.probeModeAlpha != "" {
		if err := probeModeAlpha.UnmarshalText([]byte(createConfiguration.probeModeAlpha)); err != nil {
			return fmt.Errorf("unable to parse probe mode for alpha: %w", err)
		}
	}
	if createConfiguration.probeModeBeta != "" {
		if err := probeModeBeta.UnmarshalText([]byte(createConfiguration.probeModeBeta)); err != nil {
			return fmt.Errorf("unable to parse probe mode for beta: %w", err)
		}
	}

	// Validate and convert scan mode specifications.
	var scanMode, scanModeAlpha, scanModeBeta synchronization.ScanMode
	if createConfiguration.scanMode != "" {
		if err := scanMode.UnmarshalText([]byte(createConfiguration.scanMode)); err != nil {
			return fmt.Errorf("unable to parse scan mode: %w", err)
		}
	}
	if createConfiguration.scanModeAlpha != "" {
		if err := scanModeAlpha.UnmarshalText([]byte(createConfiguration.scanModeAlpha)); err != nil {
			return fmt.Errorf("unable to parse scan mode for alpha: %w", err)
		}
	}
	if createConfiguration.scanModeBeta != "" {
		if err := scanModeBeta.UnmarshalText([]byte(createConfiguration.scanModeBeta)); err != nil {
			return fmt.Errorf("unable to parse scan mode for beta: %w", err)
		}
	}

	// Validate and convert staging mode specifications.
	var stageMode, stageModeAlpha, stageModeBeta synchronization.StageMode
	if createConfiguration.stageMode != "" {
		if err := stageMode.UnmarshalText([]byte(createConfiguration.stageMode)); err != nil {
			return fmt.Errorf("unable to parse staging mode: %w", err)
		}
	}
	if createConfiguration.stageModeAlpha != "" {
		if err := stageModeAlpha.UnmarshalText([]byte(createConfiguration.stageModeAlpha)); err != nil {
			return fmt.Errorf("unable to parse staging mode for alpha: %w", err)
		}
	}
	if createConfiguration.stageModeBeta != "" {
		if err := stageModeBeta.UnmarshalText([]byte(createConfiguration.stageModeBeta)); err != nil {
			return fmt.Errorf("unable to parse staging mode for beta: %w", err)
		}
	}

	// Validate and convert the symbolic link mode specification.
	var symbolicLinkMode core.SymbolicLinkMode
	if createConfiguration.symbolicLinkMode != "" {
		if err := symbolicLinkMode.UnmarshalText([]byte(createConfiguration.symbolicLinkMode)); err != nil {
			return fmt.Errorf("unable to parse symbolic link mode: %w", err)
		}
	}

	// Validate and convert watch mode specifications.
	var watchMode, watchModeAlpha, watchModeBeta synchronization.WatchMode
	if createConfiguration.watchMode != "" {
		if err := watchMode.UnmarshalText([]byte(createConfiguration.watchMode)); err != nil {
			return fmt.Errorf("unable to parse watch mode: %w", err)
		}
	}
	if createConfiguration.watchModeAlpha != "" {
		if err := watchModeAlpha.UnmarshalText([]byte(createConfiguration.watchModeAlpha)); err != nil {
			return fmt.Errorf("unable to parse watch mode for alpha: %w", err)
		}
	}
	if createConfiguration.watchModeBeta != "" {
		if err := watchModeBeta.UnmarshalText([]byte(createConfiguration.watchModeBeta)); err != nil {
			return fmt.Errorf("unable to parse watch mode for beta: %w", err)
		}
	}

	// There's no need to validate the watch polling intervals - any uint32
	// values are valid.

	// Validate and convert the ignore syntax specification.
	var ignoreSyntax ignore.Syntax
	if createConfiguration.ignoreSyntax != "" {
		if err := ignoreSyntax.UnmarshalText([]byte(createConfiguration.ignoreSyntax)); err != nil {
			return fmt.Errorf("unable to parse ignore syntax: %w", err)
		}
	}

	// Unfortunately we can't validate ignore specifications in any meaningful
	// way because we don't yet know the ignore syntax being used. This could be
	// specified by the global YAML configuration or (more likely) determined by
	// the default session version within the daemon. These ignores will
	// eventually be validated at endpoint initialization time, but there's no
	// convenient way to do it earlier in the session creation process.

	// Validate and convert the VCS ignore mode specification.
	var ignoreVCSMode ignore.IgnoreVCSMode
	if createConfiguration.ignoreVCS && createConfiguration.noIgnoreVCS {
		return errors.New("conflicting VCS ignore behavior specified")
	} else if createConfiguration.ignoreVCS {
		ignoreVCSMode = ignore.IgnoreVCSMode_IgnoreVCSModeIgnore
	} else if createConfiguration.noIgnoreVCS {
		ignoreVCSMode = ignore.IgnoreVCSMode_IgnoreVCSModePropagate
	}

	// Validate and convert the permissions mode specification.
	var permissionsMode core.PermissionsMode
	if createConfiguration.permissionsMode != "" {
		if err := permissionsMode.UnmarshalText([]byte(createConfiguration.permissionsMode)); err != nil {
			return fmt.Errorf("unable to parse permissions mode: %w", err)
		}
	}

	// Compute the effective permissions mode.
	// HACK: We technically don't know the daemon's default session version, so
	// we compute the default permissions mode using the default session version
	// for this executable, which (given our current distribution strategy) will
	// be the same as that of the daemon. Of course, the daemon API will
	// re-validate this, so validation here is merely best-effort and
	// informational in any case. For more information on the reasoning behind
	// this, see the note in synchronization.Version.DefaultPermissionsMode.
	effectivePermissionsMode := permissionsMode
	if effectivePermissionsMode.IsDefault() {
		effectivePermissionsMode = synchronization.DefaultVersion.DefaultPermissionsMode()
	}

	// Validate and convert default file mode specifications.
	var defaultFileMode, defaultFileModeAlpha, defaultFileModeBeta filesystem.Mode
	if createConfiguration.defaultFileMode != "" {
		if err := defaultFileMode.UnmarshalText([]byte(createConfiguration.defaultFileMode)); err != nil {
			return fmt.Errorf("unable to parse default file mode: %w", err)
		} else if err = core.EnsureDefaultFileModeValid(effectivePermissionsMode, defaultFileMode); err != nil {
			return fmt.Errorf("invalid default file mode: %w", err)
		}
	}
	if createConfiguration.defaultFileModeAlpha != "" {
		if err := defaultFileModeAlpha.UnmarshalText([]byte(createConfiguration.defaultFileModeAlpha)); err != nil {
			return fmt.Errorf("unable to parse default file mode for alpha: %w", err)
		} else if err = core.EnsureDefaultFileModeValid(effectivePermissionsMode, defaultFileModeAlpha); err != nil {
			return fmt.Errorf("invalid default file mode for alpha: %w", err)
		}
	}
	if createConfiguration.defaultFileModeBeta != "" {
		if err := defaultFileModeBeta.UnmarshalText([]byte(createConfiguration.defaultFileModeBeta)); err != nil {
			return fmt.Errorf("unable to parse default file mode for beta: %w", err)
		} else if err = core.EnsureDefaultFileModeValid(effectivePermissionsMode, defaultFileModeBeta); err != nil {
			return fmt.Errorf("invalid default file mode for beta: %w", err)
		}
	}

	// Validate and convert default directory mode specifications.
	var defaultDirectoryMode, defaultDirectoryModeAlpha, defaultDirectoryModeBeta filesystem.Mode
	if createConfiguration.defaultDirectoryMode != "" {
		if err := defaultDirectoryMode.UnmarshalText([]byte(createConfiguration.defaultDirectoryMode)); err != nil {
			return fmt.Errorf("unable to parse default directory mode: %w", err)
		} else if err = core.EnsureDefaultDirectoryModeValid(effectivePermissionsMode, defaultDirectoryMode); err != nil {
			return fmt.Errorf("invalid default directory mode: %w", err)
		}
	}
	if createConfiguration.defaultDirectoryModeAlpha != "" {
		if err := defaultDirectoryModeAlpha.UnmarshalText([]byte(createConfiguration.defaultDirectoryModeAlpha)); err != nil {
			return fmt.Errorf("unable to parse default directory mode for alpha: %w", err)
		} else if err = core.EnsureDefaultDirectoryModeValid(effectivePermissionsMode, defaultDirectoryModeAlpha); err != nil {
			return fmt.Errorf("invalid default directory mode for alpha: %w", err)
		}
	}
	if createConfiguration.defaultDirectoryModeBeta != "" {
		if err := defaultDirectoryModeBeta.UnmarshalText([]byte(createConfiguration.defaultDirectoryModeBeta)); err != nil {
			return fmt.Errorf("unable to parse default directory mode for beta: %w", err)
		} else if err = core.EnsureDefaultDirectoryModeValid(effectivePermissionsMode, defaultDirectoryModeBeta); err != nil {
			return fmt.Errorf("invalid default directory mode for beta: %w", err)
		}
	}

	// Validate default file owner specifications.
	if createConfiguration.defaultOwner != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(
			createConfiguration.defaultOwner,
		); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid ownership specification")
		}
	}
	if createConfiguration.defaultOwnerAlpha != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(
			createConfiguration.defaultOwnerAlpha,
		); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid ownership specification for alpha")
		}
	}
	if createConfiguration.defaultOwnerBeta != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(
			createConfiguration.defaultOwnerBeta,
		); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid ownership specification for beta")
		}
	}

	// Validate default file group specifications.
	if createConfiguration.defaultGroup != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(
			createConfiguration.defaultGroup,
		); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid group specification")
		}
	}
	if createConfiguration.defaultGroupAlpha != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(
			createConfiguration.defaultGroupAlpha,
		); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid group specification for alpha")
		}
	}
	if createConfiguration.defaultGroupBeta != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(
			createConfiguration.defaultGroupBeta,
		); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid group specification for beta")
		}
	}

	// Validate and convert compression algorithm specifications.
	var compressionAlgorithm, compressionAlgorithmAlpha, compressionAlgorithmBeta compression.Algorithm
	if createConfiguration.compression != "" {
		if err := compressionAlgorithm.UnmarshalText([]byte(createConfiguration.compression)); err != nil {
			return fmt.Errorf("unable to parse compression algorithm: %w", err)
		}
	}
	if createConfiguration.compressionAlpha != "" {
		if err := compressionAlgorithmAlpha.UnmarshalText([]byte(createConfiguration.compressionAlpha)); err != nil {
			return fmt.Errorf("unable to parse compression algorithm for alpha: %w", err)
		}
	}
	if createConfiguration.compressionBeta != "" {
		if err := compressionAlgorithmBeta.UnmarshalText([]byte(createConfiguration.compressionBeta)); err != nil {
			return fmt.Errorf("unable to parse compression algorithm for beta: %w", err)
		}
	}

	// Create the command line configuration and merge it into our cumulative
	// configuration.
	configuration = synchronization.MergeConfigurations(configuration, &synchronization.Configuration{
		SynchronizationMode:    synchronizationMode,
		HashingAlgorithm:       hashingAlgorithm,
		MaximumEntryCount:      createConfiguration.maximumEntryCount,
		MaximumStagingFileSize: maximumStagingFileSize,
		ProbeMode:              probeMode,
		ScanMode:               scanMode,
		StageMode:              stageMode,
		SymbolicLinkMode:       symbolicLinkMode,
		WatchMode:              watchMode,
		WatchPollingInterval:   createConfiguration.watchPollingInterval,
		IgnoreSyntax:           ignoreSyntax,
		Ignores:                createConfiguration.ignores,
		IgnoreVCSMode:          ignoreVCSMode,
		PermissionsMode:        permissionsMode,
		DefaultFileMode:        uint32(defaultFileMode),
		DefaultDirectoryMode:   uint32(defaultDirectoryMode),
		DefaultOwner:           createConfiguration.defaultOwner,
		DefaultGroup:           createConfiguration.defaultGroup,
		CompressionAlgorithm:   compressionAlgorithm,
	})

	// Create the creation specification.
	specification := &synchronizationsvc.CreationSpecification{
		Alpha:         alpha,
		Beta:          beta,
		Configuration: configuration,
		ConfigurationAlpha: &synchronization.Configuration{
			ProbeMode:            probeModeAlpha,
			ScanMode:             scanModeAlpha,
			StageMode:            stageModeAlpha,
			WatchMode:            watchModeAlpha,
			WatchPollingInterval: createConfiguration.watchPollingIntervalAlpha,
			DefaultFileMode:      uint32(defaultFileModeAlpha),
			DefaultDirectoryMode: uint32(defaultDirectoryModeAlpha),
			DefaultOwner:         createConfiguration.defaultOwnerAlpha,
			DefaultGroup:         createConfiguration.defaultGroupAlpha,
			CompressionAlgorithm: compressionAlgorithmAlpha,
		},
		ConfigurationBeta: &synchronization.Configuration{
			ProbeMode:            probeModeBeta,
			ScanMode:             scanModeBeta,
			StageMode:            stageModeBeta,
			WatchMode:            watchModeBeta,
			WatchPollingInterval: createConfiguration.watchPollingIntervalBeta,
			DefaultFileMode:      uint32(defaultFileModeBeta),
			DefaultDirectoryMode: uint32(defaultDirectoryModeBeta),
			DefaultOwner:         createConfiguration.defaultOwnerBeta,
			DefaultGroup:         createConfiguration.defaultGroupBeta,
			CompressionAlgorithm: compressionAlgorithmBeta,
		},
		Name:   createConfiguration.name,
		Labels: labels,
		Paused: createConfiguration.paused,
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.Connect(true, true)
	if err != nil {
		return fmt.Errorf("unable to connect to daemon: %w", err)
	}
	defer daemonConnection.Close()

	// Perform the create operation.
	identifier, err := CreateWithSpecification(daemonConnection, specification)
	if err != nil {
		return err
	}

	// Print the session identifier.
	fmt.Println("Created session", identifier)

	// Success.
	return nil
}

// createCommand is the create command.
var createCommand = &cobra.Command{
	Use:          "create <alpha> <beta>",
	Short:        "Create and start a new synchronization session",
	RunE:         createMain,
	SilenceUsage: true,
}

// createConfiguration stores configuration for the create command.
var createConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
	// name is the name specification for the session.
	name string
	// labels are the label specifications for the session.
	labels []string
	// paused indicates whether or not to create the session in a pre-paused
	// state.
	paused bool
	// noGlobalConfiguration specifies whether or not the global configuration
	// file should be ignored.
	noGlobalConfiguration bool
	// configurationFiles stores paths of additional files from which to load
	// default configuration.
	configurationFiles []string
	// synchronizationMode specifies the synchronization mode for the session.
	synchronizationMode string
	// hash specifies the hashing algorithm to use for the session.
	hash string
	// maximumEntryCount specifies the maximum number of filesystem entries that
	// endpoints will tolerate managing.
	maximumEntryCount uint64
	// maximumStagingFileSize is the maximum file size that endpoints will
	// stage. It can be specified in human-friendly units.
	maximumStagingFileSize string
	// probeMode specifies the filesystem probing mode to use for the session.
	probeMode string
	// probeModeAlpha specifies the filesystem probing mode to use for the
	// session, taking priority over probeMode on alpha if specified.
	probeModeAlpha string
	// probeModeBeta specifies the filesystem probing mode to use for the
	// session, taking priority over probeMode on beta if specified.
	probeModeBeta string
	// scanMode specifies the scan mode to use for the session.
	scanMode string
	// scanModeAlpha specifies the scan mode to use for the session, taking
	// priority over scanMode on alpha if specified.
	scanModeAlpha string
	// scanModeBeta specifies the scan mode to use for the session, taking
	// priority over scanMode on beta if specified.
	scanModeBeta string
	// stageMode specifies the file staging mode to use for the session.
	stageMode string
	// stageModeAlpha specifies the file staging mode to use for the session,
	// taking priority over stageMode on alpha if specified.
	stageModeAlpha string
	// stageModeBeta specifies the file staging mode to use for the session,
	// taking priority over stageMode on beta if specified.
	stageModeBeta string
	// symbolicLinkMode specifies the symbolic link handling mode to use for
	// the session.
	symbolicLinkMode string
	// watchMode specifies the filesystem watching mode to use for the session.
	watchMode string
	// watchModeAlpha specifies the filesystem watching mode to use for the
	// session, taking priority over watchMode on alpha if specified.
	watchModeAlpha string
	// watchModeBeta specifies the filesystem watching mode to use for the
	// session, taking priority over watchMode on beta if specified.
	watchModeBeta string
	// watchPollingInterval specifies the polling interval to use if using
	// poll-based or hybrid watching.
	watchPollingInterval uint32
	// watchPollingIntervalAlpha specifies the polling interval to use if using
	// poll-based or hybrid watching, taking priority over watchPollingInterval
	// on alpha if specified.
	watchPollingIntervalAlpha uint32
	// watchPollingIntervalBeta specifies the polling interval to use if using
	// poll-based or hybrid watching, taking priority over watchPollingInterval
	// on beta if specified.
	watchPollingIntervalBeta uint32
	// ignoreSyntax specifies the ignore syntax and semantics for the session.
	ignoreSyntax string
	// ignores is the list of ignore specifications for the session.
	ignores []string
	// ignoreVCS specifies whether or not to enable VCS ignores for the session.
	ignoreVCS bool
	// noIgnoreVCS specifies whether or not to disable VCS ignores for the
	// session.
	noIgnoreVCS bool
	// permissionsMode specifies the permissions mode to use for the session.
	permissionsMode string
	// defaultFileMode specifies the default permission mode to use for new
	// files in "portable" permission propagation mode, with endpoint-specific
	// specifications taking priority.
	defaultFileMode string
	// defaultFileModeAlpha specifies the default permission mode to use for new
	// files on alpha in "portable" permission propagation mode, taking priority
	// over defaultFileMode on alpha if specified.
	defaultFileModeAlpha string
	// defaultFileModeBeta specifies the default permission mode to use for new
	// files on beta in "portable" permission propagation mode, taking priority
	// over defaultFileMode on beta if specified.
	defaultFileModeBeta string
	// defaultDirectoryMode specifies the default permission mode to use for new
	// directories in "portable" permission propagation mode, with endpoint-
	// specific specifications taking priority.
	defaultDirectoryMode string
	// defaultDirectoryModeAlpha specifies the default permission mode to use
	// for new directories on alpha in "portable" permission propagation mode,
	// taking priority over defaultDirectoryMode on alpha if specified.
	defaultDirectoryModeAlpha string
	// defaultDirectoryModeBeta specifies the default permission mode to use for
	// new directories on beta in "portable" permission propagation mode, taking
	// priority over defaultDirectoryMode on beta if specified.
	defaultDirectoryModeBeta string
	// defaultOwner specifies the default owner identifier to use when setting
	// ownership of new files and directories in "portable" permission
	// propagation mode, with endpoint-specific specifications taking priority.
	defaultOwner string
	// defaultOwnerAlpha specifies the default owner identifier to use when
	// setting ownership of new files and directories on alpha in "portable"
	// permission propagation mode, taking priority over defaultOwner on alpha
	// if specified.
	defaultOwnerAlpha string
	// defaultOwnerBeta specifies the default owner identifier to use when
	// setting ownership of new files and directories on beta in "portable"
	// permission propagation mode, taking priority over defaultOwner on beta if
	// specified.
	defaultOwnerBeta string
	// defaultGroup specifies the default group identifier to use when setting
	// ownership of new files and directories in "portable" permission
	// propagation mode, with endpoint-specific specifications taking priority.
	defaultGroup string
	// defaultGroupAlpha specifies the default group identifier to use when
	// setting ownership of new files and directories on alpha in "portable"
	// permission propagation mode, taking priority over defaultGroup on alpha
	// if specified.
	defaultGroupAlpha string
	// defaultGroupBeta specifies the default group identifier to use when
	// setting ownership of new files and directories on beta in "portable"
	// permission propagation mode, taking priority over defaultGroup on beta if
	// specified.
	defaultGroupBeta string
	// compression specifies the compression algorithm to use when communicating
	// with remote endpoints.
	compression string
	// compressionAlpha specifies the compression algorithm to use when
	// communicating with a remote alpha endpoint.
	compressionAlpha string
	// compressionBeta specifies the compression algorithm to use when
	// communicating with a remote beta endpoint.
	compressionBeta string
}

func init() {
	// Grab a handle for the command line flags.
	flags := createCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&createConfiguration.help, "help", "h", false, "Show help information")

	// Wire up name and label flags.
	flags.StringVarP(&createConfiguration.name, "name", "n", "", "Specify a name for the session")
	flags.StringSliceVarP(&createConfiguration.labels, "label", "l", nil, "Specify labels")

	// Wire up paused flags.
	flags.BoolVarP(&createConfiguration.paused, "paused", "p", false, "Create the session pre-paused")

	// Wire up general configuration flags.
	flags.BoolVar(&createConfiguration.noGlobalConfiguration, "no-global-configuration", false, "Ignore the global configuration file")
	flags.StringSliceVarP(&createConfiguration.configurationFiles, "configuration-file", "c", nil, "Specify additional files from which to load (and merge) default configuration parameters")

	// Wire up synchronization flags.
	flags.StringVarP(&createConfiguration.synchronizationMode, "mode", "m", "", "Specify synchronization mode (two-way-safe|two-way-resolved|one-way-safe|one-way-replica)")
	flags.StringVarP(&createConfiguration.hash, "hash", "H", "", "Specify content hashing algorithm ("+hashFlagOptions+")")
	flags.Uint64Var(&createConfiguration.maximumEntryCount, "max-entry-count", 0, "Specify the maximum number of entries that endpoints will manage")
	flags.StringVar(&createConfiguration.maximumStagingFileSize, "max-staging-file-size", "", "Specify the maximum (individual) file size that endpoints will stage")
	flags.StringVar(&createConfiguration.probeMode, "probe-mode", "", "Specify probe mode (probe|assume)")
	flags.StringVar(&createConfiguration.probeModeAlpha, "probe-mode-alpha", "", "Specify probe mode for alpha (probe|assume)")
	flags.StringVar(&createConfiguration.probeModeBeta, "probe-mode-beta", "", "Specify probe mode for beta (probe|assume)")
	flags.StringVar(&createConfiguration.scanMode, "scan-mode", "", "Specify scan mode (full|accelerated)")
	flags.StringVar(&createConfiguration.scanModeAlpha, "scan-mode-alpha", "", "Specify scan mode for alpha (full|accelerated)")
	flags.StringVar(&createConfiguration.scanModeBeta, "scan-mode-beta", "", "Specify scan mode for beta (full|accelerated)")
	flags.StringVar(&createConfiguration.stageMode, "stage-mode", "", "Specify staging mode (mutagen|neighboring)")
	flags.StringVar(&createConfiguration.stageModeAlpha, "stage-mode-alpha", "", "Specify staging mode for alpha (mutagen|neighboring)")
	flags.StringVar(&createConfiguration.stageModeBeta, "stage-mode-beta", "", "Specify staging mode for beta (mutagen|neighboring)")

	// Wire up symbolic link flags.
	flags.StringVar(&createConfiguration.symbolicLinkMode, "symlink-mode", "", "Specify symlink mode (ignore|portable|posix-raw)")

	// Wire up watch flags.
	flags.StringVar(&createConfiguration.watchMode, "watch-mode", "", "Specify watch mode (portable|force-poll|no-watch)")
	flags.StringVar(&createConfiguration.watchModeAlpha, "watch-mode-alpha", "", "Specify watch mode for alpha (portable|force-poll|no-watch)")
	flags.StringVar(&createConfiguration.watchModeBeta, "watch-mode-beta", "", "Specify watch mode for beta (portable|force-poll|no-watch)")
	flags.Uint32Var(&createConfiguration.watchPollingInterval, "watch-polling-interval", 0, "Specify watch polling interval in seconds")
	flags.Uint32Var(&createConfiguration.watchPollingIntervalAlpha, "watch-polling-interval-alpha", 0, "Specify watch polling interval in seconds for alpha")
	flags.Uint32Var(&createConfiguration.watchPollingIntervalBeta, "watch-polling-interval-beta", 0, "Specify watch polling interval in seconds for beta")

	// Wire up ignore flags.
	flags.StringVar(&createConfiguration.ignoreSyntax, "ignore-syntax", "", "Specify ignore syntax (mutagen|docker)")
	flags.StringSliceVarP(&createConfiguration.ignores, "ignore", "i", nil, "Specify ignore paths")
	flags.BoolVar(&createConfiguration.ignoreVCS, "ignore-vcs", false, "Ignore VCS directories")
	flags.BoolVar(&createConfiguration.noIgnoreVCS, "no-ignore-vcs", false, "Propagate VCS directories")

	// Wire up permission flags.
	flags.StringVar(&createConfiguration.permissionsMode, "permissions-mode", "", "Specify permissions mode (portable|manual)")
	flags.StringVar(&createConfiguration.defaultFileMode, "default-file-mode", "", "Specify default file permission mode")
	flags.StringVar(&createConfiguration.defaultFileModeAlpha, "default-file-mode-alpha", "", "Specify default file permission mode for alpha")
	flags.StringVar(&createConfiguration.defaultFileModeBeta, "default-file-mode-beta", "", "Specify default file permission mode for beta")
	flags.StringVar(&createConfiguration.defaultDirectoryMode, "default-directory-mode", "", "Specify default directory permission mode")
	flags.StringVar(&createConfiguration.defaultDirectoryModeAlpha, "default-directory-mode-alpha", "", "Specify default directory permission mode for alpha")
	flags.StringVar(&createConfiguration.defaultDirectoryModeBeta, "default-directory-mode-beta", "", "Specify default directory permission mode for beta")
	flags.StringVar(&createConfiguration.defaultOwner, "default-owner", "", "Specify default file/directory owner")
	flags.StringVar(&createConfiguration.defaultOwnerAlpha, "default-owner-alpha", "", "Specify default file/directory owner for alpha")
	flags.StringVar(&createConfiguration.defaultOwnerBeta, "default-owner-beta", "", "Specify default file/directory owner for beta")
	flags.StringVar(&createConfiguration.defaultGroup, "default-group", "", "Specify default file/directory group")
	flags.StringVar(&createConfiguration.defaultGroupAlpha, "default-group-alpha", "", "Specify default file/directory group for alpha")
	flags.StringVar(&createConfiguration.defaultGroupBeta, "default-group-beta", "", "Specify default file/directory group for beta")

	// Wire up compression flags.
	flags.StringVarP(&createConfiguration.compression, "compression", "C", "", "Specify compression algorithm ("+compressionFlagOptions+")")
	flags.StringVar(&createConfiguration.compressionAlpha, "compression-alpha", "", "Specify compression algorithm for alpha ("+compressionFlagOptions+")")
	flags.StringVar(&createConfiguration.compressionBeta, "compression-beta", "", "Specify compression algorithm for beta ("+compressionFlagOptions+")")

	// Set up flag normalization. This is only required to handle aliases.
	flags.SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		if name == "sync-mode" {
			name = "mode"
		}
		return pflag.NormalizedName(name)
	})
}
