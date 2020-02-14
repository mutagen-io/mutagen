package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/dustin/go-humanize"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"
	"github.com/mutagen-io/mutagen/pkg/configuration/global"
	"github.com/mutagen-io/mutagen/pkg/configuration/legacy"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/filesystem/behavior"
	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	"github.com/mutagen-io/mutagen/pkg/prompting"
	"github.com/mutagen-io/mutagen/pkg/selection"
	synchronizationsvc "github.com/mutagen-io/mutagen/pkg/service/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
	"github.com/mutagen-io/mutagen/pkg/url"
)

// loadAndValidateGlobalSynchronizationConfiguration loads a YAML-based global
// configuration, extracts the synchronization component, and converts it to a
// Protocol Buffers session configuration, and validates it.
func loadAndValidateGlobalSynchronizationConfiguration(path string) (*synchronization.Configuration, error) {
	// Load the YAML configuration.
	yamlConfiguration, err := global.LoadConfiguration(path)
	if err != nil {
		return nil, err
	}

	// Convert the YAML configuration to a Protocol Buffers representation and
	// validate it.
	configuration := yamlConfiguration.Synchronization.Defaults.Configuration()
	if err := configuration.EnsureValid(false); err != nil {
		return nil, errors.Wrap(err, "invalid configuration")
	}

	// Success.
	return configuration, nil
}

// loadAndValidateLegacyTOMLConfiguration loads a legacy TOML-based
// configuration, converts it to a Protocol Buffers session configuration, and
// validates it.
func loadAndValidateLegacyTOMLConfiguration(path string) (*synchronization.Configuration, error) {
	// Load the TOML configuration.
	tomlConfiguration, err := legacy.LoadConfiguration(path)
	if err != nil {
		return nil, err
	}

	// Convert the TOML configuration to a Protocol Buffers representation and
	// validate it.
	configuration := tomlConfiguration.Configuration()
	if err := configuration.EnsureValid(false); err != nil {
		return nil, errors.Wrap(err, "invalid configuration")
	}

	// Success.
	return configuration, nil
}

// CreateWithSpecification is an orchestration convenience method invokes the
// creation using the provided session specification. Unlike other orchestration
// methods, it requires provision of a client to avoid creating one for each
// request.
func CreateWithSpecification(
	service synchronizationsvc.SynchronizationClient,
	specification *synchronizationsvc.CreationSpecification,
) (string, error) {
	// Invoke the session create method. The stream will close when the
	// associated context is cancelled.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := service.Create(ctx)
	if err != nil {
		return "", errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to invoke create")
	}

	// Send the initial request.
	request := &synchronizationsvc.CreateRequest{Specification: specification}
	if err := stream.Send(request); err != nil {
		return "", errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send create request")
	}

	// Create a status line printer and defer a break.
	statusLinePrinter := &cmd.StatusLinePrinter{}
	defer statusLinePrinter.BreakIfNonEmpty()

	// Receive and process responses until we're done.
	for {
		if response, err := stream.Recv(); err != nil {
			return "", errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "create failed")
		} else if err = response.EnsureValid(); err != nil {
			return "", errors.Wrap(err, "invalid create response received")
		} else if response.Session != "" {
			statusLinePrinter.Print(fmt.Sprintf("Created session %s", response.Session))
			return response.Session, nil
		} else if response.Message != "" {
			statusLinePrinter.Print(response.Message)
			if err := stream.Send(&synchronizationsvc.CreateRequest{}); err != nil {
				return "", errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send message response")
			}
		} else if response.Prompt != "" {
			statusLinePrinter.BreakIfNonEmpty()
			if response, err := prompting.PromptCommandLine(response.Prompt); err != nil {
				return "", errors.Wrap(err, "unable to perform prompting")
			} else if err = stream.Send(&synchronizationsvc.CreateRequest{Response: response}); err != nil {
				return "", errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send prompt response")
			}
		}
	}
}

func createMain(command *cobra.Command, arguments []string) error {
	// Validate, extract, and parse URLs.
	if len(arguments) != 2 {
		return errors.New("invalid number of endpoint URLs provided")
	}
	alpha, err := url.Parse(arguments[0], url.Kind_Synchronization, true)
	if err != nil {
		return errors.Wrap(err, "unable to parse alpha URL")
	}
	beta, err := url.Parse(arguments[1], url.Kind_Synchronization, false)
	if err != nil {
		return errors.Wrap(err, "unable to parse beta URL")
	}

	// Validate the name.
	if err := selection.EnsureNameValid(createConfiguration.name); err != nil {
		return errors.Wrap(err, "invalid session name")
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
			return errors.Wrap(err, "invalid label key")
		} else if err := selection.EnsureLabelValueValid(value); err != nil {
			return errors.Wrap(err, "invalid label value")
		}
		labels[key] = value
	}

	// Create a default session configuration which will form the basis of our
	// cumulative configuration.
	configuration := &synchronization.Configuration{}

	// Unless disabled, load configuration from the global configuration file
	// and merge it into our cumulative configuration.
	var globalConfigurationNonExistent bool
	if !createConfiguration.noGlobalConfiguration {
		// Compute the path to the global configuration file.
		globalConfigurationPath, err := global.ConfigurationPath()
		if err != nil {
			return errors.Wrap(err, "unable to compute path to global configuration file")
		}

		// Attempt to load the file. We allow it to not exist.
		globalConfiguration, err := loadAndValidateGlobalSynchronizationConfiguration(globalConfigurationPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return errors.Wrap(err, "unable to load global configuration")
			}
			globalConfigurationNonExistent = true
		} else {
			configuration = synchronization.MergeConfigurations(configuration, globalConfiguration)
		}
	}

	// If we tried to load the global configuration and it didn't exist, then
	// try to load the legacy global configuration.
	if globalConfigurationNonExistent {
		// Compute the path to the global configuration file.
		legacyGlobalConfigurationPath, err := legacy.ConfigurationPath()
		if err != nil {
			return errors.Wrap(err, "unable to compute path to legacy global configuration file")
		}

		// Attempt to load the file. We don't require that the legacy global
		// configuration exist, but if it does (and since the YAML-based global
		// configuration file didn't exist), then we warn the user about its
		// deprecation.
		globalConfiguration, err := loadAndValidateLegacyTOMLConfiguration(legacyGlobalConfigurationPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return errors.Wrap(err, "unable to load legacy global configuration")
			}
		} else {
			configuration = synchronization.MergeConfigurations(configuration, globalConfiguration)
			cmd.Warning("TOML-based global configuration files are deprecated, please migrate to YAML")
		}
	}

	// If a configuration file has been specified, then load it and merge it
	// into our cumulative configuration. We handle its loading based on the
	// extension, warning if a legacy TOML configuration file is used.
	if createConfiguration.configurationFile != "" {
		if filepath.Ext(createConfiguration.configurationFile) == ".toml" {
			cmd.Warning("TOML-based configuration files are deprecated, please migrate to YAML")
			if c, err := loadAndValidateLegacyTOMLConfiguration(createConfiguration.configurationFile); err != nil {
				return errors.Wrap(err, "unable to load legacy configuration file")
			} else {
				configuration = synchronization.MergeConfigurations(configuration, c)
			}
		} else {
			if c, err := loadAndValidateGlobalSynchronizationConfiguration(createConfiguration.configurationFile); err != nil {
				return errors.Wrap(err, "unable to load configuration file")
			} else {
				configuration = synchronization.MergeConfigurations(configuration, c)
			}
		}
	}

	// Validate and convert the synchronization mode specification.
	var synchronizationMode core.SynchronizationMode
	if createConfiguration.synchronizationMode != "" {
		if err := synchronizationMode.UnmarshalText([]byte(createConfiguration.synchronizationMode)); err != nil {
			return errors.Wrap(err, "unable to parse synchronization mode")
		}
	}

	// There's no need to validate the maximum entry count - any uint64 value is
	// valid.

	// Validate and convert the maximum staging file size.
	var maximumStagingFileSize uint64
	if createConfiguration.maximumStagingFileSize != "" {
		if s, err := humanize.ParseBytes(createConfiguration.maximumStagingFileSize); err != nil {
			return errors.Wrap(err, "unable to parse maximum staging file size")
		} else {
			maximumStagingFileSize = s
		}
	}

	// Validate and convert probe mode specifications.
	var probeMode, probeModeAlpha, probeModeBeta behavior.ProbeMode
	if createConfiguration.probeMode != "" {
		if err := probeMode.UnmarshalText([]byte(createConfiguration.probeMode)); err != nil {
			return errors.Wrap(err, "unable to parse probe mode")
		}
	}
	if createConfiguration.probeModeAlpha != "" {
		if err := probeModeAlpha.UnmarshalText([]byte(createConfiguration.probeModeAlpha)); err != nil {
			return errors.Wrap(err, "unable to parse probe mode for alpha")
		}
	}
	if createConfiguration.probeModeBeta != "" {
		if err := probeModeBeta.UnmarshalText([]byte(createConfiguration.probeModeBeta)); err != nil {
			return errors.Wrap(err, "unable to parse probe mode for beta")
		}
	}

	// Validate and convert scan mode specifications.
	var scanMode, scanModeAlpha, scanModeBeta synchronization.ScanMode
	if createConfiguration.scanMode != "" {
		if err := scanMode.UnmarshalText([]byte(createConfiguration.scanMode)); err != nil {
			return errors.Wrap(err, "unable to parse scan mode")
		}
	}
	if createConfiguration.scanModeAlpha != "" {
		if err := scanModeAlpha.UnmarshalText([]byte(createConfiguration.scanModeAlpha)); err != nil {
			return errors.Wrap(err, "unable to parse scan mode for alpha")
		}
	}
	if createConfiguration.scanModeBeta != "" {
		if err := scanModeBeta.UnmarshalText([]byte(createConfiguration.scanModeBeta)); err != nil {
			return errors.Wrap(err, "unable to parse scan mode for beta")
		}
	}

	// Validate and convert staging mode specifications.
	var stageMode, stageModeAlpha, stageModeBeta synchronization.StageMode
	if createConfiguration.stageMode != "" {
		if err := stageMode.UnmarshalText([]byte(createConfiguration.stageMode)); err != nil {
			return errors.Wrap(err, "unable to parse staging mode")
		}
	}
	if createConfiguration.stageModeAlpha != "" {
		if err := stageModeAlpha.UnmarshalText([]byte(createConfiguration.stageModeAlpha)); err != nil {
			return errors.Wrap(err, "unable to parse staging mode for alpha")
		}
	}
	if createConfiguration.stageModeBeta != "" {
		if err := stageModeBeta.UnmarshalText([]byte(createConfiguration.stageModeBeta)); err != nil {
			return errors.Wrap(err, "unable to parse staging mode for beta")
		}
	}

	// Validate and convert the symbolic link mode specification.
	var symbolicLinkMode core.SymlinkMode
	if createConfiguration.symbolicLinkMode != "" {
		if err := symbolicLinkMode.UnmarshalText([]byte(createConfiguration.symbolicLinkMode)); err != nil {
			return errors.Wrap(err, "unable to parse symbolic link mode")
		}
	}

	// Validate and convert watch mode specifications.
	var watchMode, watchModeAlpha, watchModeBeta synchronization.WatchMode
	if createConfiguration.watchMode != "" {
		if err := watchMode.UnmarshalText([]byte(createConfiguration.watchMode)); err != nil {
			return errors.Wrap(err, "unable to parse watch mode")
		}
	}
	if createConfiguration.watchModeAlpha != "" {
		if err := watchModeAlpha.UnmarshalText([]byte(createConfiguration.watchModeAlpha)); err != nil {
			return errors.Wrap(err, "unable to parse watch mode for alpha")
		}
	}
	if createConfiguration.watchModeBeta != "" {
		if err := watchModeBeta.UnmarshalText([]byte(createConfiguration.watchModeBeta)); err != nil {
			return errors.Wrap(err, "unable to parse watch mode for beta")
		}
	}

	// There's no need to validate the watch polling intervals - any uint32
	// values are valid.

	// Validate ignore specifications.
	for _, ignore := range createConfiguration.ignores {
		if !core.ValidIgnorePattern(ignore) {
			return errors.Errorf("invalid ignore pattern: %s", ignore)
		}
	}

	// Validate and convert the VCS ignore mode specification.
	var ignoreVCSMode core.IgnoreVCSMode
	if createConfiguration.ignoreVCS && createConfiguration.noIgnoreVCS {
		return errors.New("conflicting VCS ignore behavior specified")
	} else if createConfiguration.ignoreVCS {
		ignoreVCSMode = core.IgnoreVCSMode_IgnoreVCSModeIgnore
	} else if createConfiguration.noIgnoreVCS {
		ignoreVCSMode = core.IgnoreVCSMode_IgnoreVCSModePropagate
	}

	// Validate and convert default file mode specifications.
	var defaultFileMode, defaultFileModeAlpha, defaultFileModeBeta filesystem.Mode
	if createConfiguration.defaultFileMode != "" {
		if err := defaultFileMode.UnmarshalText([]byte(createConfiguration.defaultFileMode)); err != nil {
			return errors.Wrap(err, "unable to parse default file mode")
		} else if err = core.EnsureDefaultFileModeValid(defaultFileMode); err != nil {
			return errors.Wrap(err, "invalid default file mode")
		}
	}
	if createConfiguration.defaultFileModeAlpha != "" {
		if err := defaultFileModeAlpha.UnmarshalText([]byte(createConfiguration.defaultFileModeAlpha)); err != nil {
			return errors.Wrap(err, "unable to parse default file mode for alpha")
		} else if err = core.EnsureDefaultFileModeValid(defaultFileModeAlpha); err != nil {
			return errors.Wrap(err, "invalid default file mode for alpha")
		}
	}
	if createConfiguration.defaultFileModeBeta != "" {
		if err := defaultFileModeBeta.UnmarshalText([]byte(createConfiguration.defaultFileModeBeta)); err != nil {
			return errors.Wrap(err, "unable to parse default file mode for beta")
		} else if err = core.EnsureDefaultFileModeValid(defaultFileModeBeta); err != nil {
			return errors.Wrap(err, "invalid default file mode for beta")
		}
	}

	// Validate and convert default directory mode specifications.
	var defaultDirectoryMode, defaultDirectoryModeAlpha, defaultDirectoryModeBeta filesystem.Mode
	if createConfiguration.defaultDirectoryMode != "" {
		if err := defaultDirectoryMode.UnmarshalText([]byte(createConfiguration.defaultDirectoryMode)); err != nil {
			return errors.Wrap(err, "unable to parse default directory mode")
		} else if err = core.EnsureDefaultDirectoryModeValid(defaultDirectoryMode); err != nil {
			return errors.Wrap(err, "invalid default directory mode")
		}
	}
	if createConfiguration.defaultDirectoryModeAlpha != "" {
		if err := defaultDirectoryModeAlpha.UnmarshalText([]byte(createConfiguration.defaultDirectoryModeAlpha)); err != nil {
			return errors.Wrap(err, "unable to parse default directory mode for alpha")
		} else if err = core.EnsureDefaultDirectoryModeValid(defaultDirectoryModeAlpha); err != nil {
			return errors.Wrap(err, "invalid default directory mode for alpha")
		}
	}
	if createConfiguration.defaultDirectoryModeBeta != "" {
		if err := defaultDirectoryModeBeta.UnmarshalText([]byte(createConfiguration.defaultDirectoryModeBeta)); err != nil {
			return errors.Wrap(err, "unable to parse default directory mode for beta")
		} else if err = core.EnsureDefaultDirectoryModeValid(defaultDirectoryModeBeta); err != nil {
			return errors.Wrap(err, "invalid default directory mode for beta")
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

	// Create the command line configuration and merge it into our cumulative
	// configuration.
	configuration = synchronization.MergeConfigurations(configuration, &synchronization.Configuration{
		SynchronizationMode:    synchronizationMode,
		MaximumEntryCount:      createConfiguration.maximumEntryCount,
		MaximumStagingFileSize: maximumStagingFileSize,
		ProbeMode:              probeMode,
		ScanMode:               scanMode,
		StageMode:              stageMode,
		SymlinkMode:            symbolicLinkMode,
		WatchMode:              watchMode,
		WatchPollingInterval:   createConfiguration.watchPollingInterval,
		Ignores:                createConfiguration.ignores,
		IgnoreVCSMode:          ignoreVCSMode,
		DefaultFileMode:        uint32(defaultFileMode),
		DefaultDirectoryMode:   uint32(defaultDirectoryMode),
		DefaultOwner:           createConfiguration.defaultOwner,
		DefaultGroup:           createConfiguration.defaultGroup,
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
		},
		Name:   createConfiguration.name,
		Labels: labels,
		Paused: createConfiguration.paused,
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.CreateClientConnection(true, true)
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Create a synchronization service client.
	service := synchronizationsvc.NewSynchronizationClient(daemonConnection)

	// Perform creation.
	_, err = CreateWithSpecification(service, specification)
	return err
}

var createCommand = &cobra.Command{
	Use:          "create <alpha> <beta>",
	Short:        "Create and start a new synchronization session",
	RunE:         createMain,
	SilenceUsage: true,
}

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
	// configurationFile specifies a file from which to load configuration. It
	// should be a path relative to the working directory.
	configurationFile string
	// synchronizationMode specifies the synchronization mode for the session.
	synchronizationMode string
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
	// ignores is the list of ignore specifications for the session.
	ignores []string
	// ignoreVCS specifies whether or not to enable VCS ignores for the session.
	ignoreVCS bool
	// noIgnoreVCS specifies whether or not to disable VCS ignores for the
	// session.
	noIgnoreVCS bool
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
	flags.StringVarP(&createConfiguration.configurationFile, "configuration-file", "c", "", "Specify a file from which to load session configuration")

	// Wire up synchronization flags.
	flags.StringVarP(&createConfiguration.synchronizationMode, "sync-mode", "m", "", "Specify synchronization mode (two-way-safe|two-way-resolved|one-way-safe|one-way-replica)")
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
	flags.StringSliceVarP(&createConfiguration.ignores, "ignore", "i", nil, "Specify ignore paths")
	flags.BoolVar(&createConfiguration.ignoreVCS, "ignore-vcs", false, "Ignore VCS directories")
	flags.BoolVar(&createConfiguration.noIgnoreVCS, "no-ignore-vcs", false, "Propagate VCS directories")

	// Wire up permission flags.
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
}
