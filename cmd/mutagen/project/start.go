package project

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"
	"github.com/mutagen-io/mutagen/cmd/mutagen/forward"
	"github.com/mutagen-io/mutagen/cmd/mutagen/sync"

	"github.com/mutagen-io/mutagen/pkg/configuration/global"
	"github.com/mutagen-io/mutagen/pkg/filesystem/locking"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/identifier"
	"github.com/mutagen-io/mutagen/pkg/project"
	"github.com/mutagen-io/mutagen/pkg/selection"
	forwardingsvc "github.com/mutagen-io/mutagen/pkg/service/forwarding"
	synchronizationsvc "github.com/mutagen-io/mutagen/pkg/service/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/url"
)

// startMain is the entry point for the start command.
func startMain(_ *cobra.Command, _ []string) error {
	// Compute the name of the configuration file and ensure that our working
	// directory is that in which the file resides. This is required for
	// relative paths (including relative synchronization paths and relative
	// Unix Domain Socket paths) to be resolved relative to the project
	// configuration file.
	configurationFileName := project.DefaultConfigurationFileName
	if startConfiguration.projectFile != "" {
		var directory string
		directory, configurationFileName = filepath.Split(startConfiguration.projectFile)
		if directory != "" {
			if err := os.Chdir(directory); err != nil {
				return fmt.Errorf("unable to switch to target directory: %w", err)
			}
		}
	}

	// Compute the lock path.
	lockPath := configurationFileName + project.LockFileExtension

	// Track whether or not we should remove the lock file on return.
	var removeLockFileOnReturn bool

	// Create a locker and defer its closure and potential removal. On Windows
	// systems, we have to handle this removal after the file is closed.
	locker, err := locking.NewLocker(lockPath, 0600)
	if err != nil {
		return fmt.Errorf("unable to create project locker: %w", err)
	}
	defer func() {
		locker.Close()
		if removeLockFileOnReturn && runtime.GOOS == "windows" {
			os.Remove(lockPath)
		}
	}()

	// Acquire the project lock and defer its release and potential removal. On
	// Windows systems, we can't remove the lock file if it's locked or even
	// just opened, so we handle removal for Windows systems after we close the
	// lock file (see above). In this case, we truncate the lock file before
	// releasing it to ensure that any other process that opens or acquires the
	// lock file before we manage to remove it will simply see an empty lock
	// file, which it will ignore or attempt to remove.
	if err := locker.Lock(true); err != nil {
		return fmt.Errorf("unable to acquire project lock: %w", err)
	}
	defer func() {
		if removeLockFileOnReturn {
			if runtime.GOOS == "windows" {
				locker.Truncate(0)
			} else {
				os.Remove(lockPath)
			}
		}
		locker.Unlock()
	}()

	// Read the full contents of the lock file and ensure that it's empty.
	buffer := &bytes.Buffer{}
	if length, err := buffer.ReadFrom(locker); err != nil {
		return fmt.Errorf("unable to read project lock: %w", err)
	} else if length != 0 {
		return errors.New("project already running")
	}

	// At this point we know that there was no previous project running, but we
	// haven't yet created any resources, so defer removal of the lock file that
	// we've created in case we run into any errors loading configuration
	// information.
	removeLockFileOnReturn = true

	// Create a unique project identifier.
	identifier, err := identifier.New(identifier.PrefixProject)
	if err != nil {
		return fmt.Errorf("unable to generate project identifier: %w", err)
	}

	// Write the project identifier to the lock file.
	if _, err := locker.Write([]byte(identifier)); err != nil {
		return fmt.Errorf("unable to write project identifier: %w", err)
	}

	// Load the configuration file.
	configuration, err := project.LoadConfiguration(configurationFileName)
	if err != nil {
		return fmt.Errorf("unable to load configuration file: %w", err)
	}

	// Unless disabled, attempt to load configuration from the global
	// configuration file and use it as the base for our core session
	// configurations.
	globalConfigurationForwarding := &forwarding.Configuration{}
	globalConfigurationSynchronization := &synchronization.Configuration{}
	if !startConfiguration.noGlobalConfiguration {
		// Compute the path to the global configuration file.
		globalConfigurationPath, err := global.ConfigurationPath()
		if err != nil {
			return fmt.Errorf("unable to compute path to global configuration file: %w", err)
		}

		// Attempt to load and validate the file. We allow it to not exist.
		globalConfiguration, err := global.LoadConfiguration(globalConfigurationPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("unable to load global configuration: %w", err)
			}
		} else {
			globalConfigurationForwarding = globalConfiguration.Forwarding.Defaults.ToInternalConfiguration()
			if err := globalConfigurationForwarding.EnsureValid(false); err != nil {
				return fmt.Errorf("invalid global forwarding configuration: %w", err)
			}
			globalConfigurationSynchronization = globalConfiguration.Synchronization.Defaults.ToInternalConfiguration()
			if err := globalConfigurationSynchronization.EnsureValid(false); err != nil {
				return fmt.Errorf("invalid global synchronization configuration: %w", err)
			}
		}
	}

	// Extract and validate forwarding defaults.
	var defaultSource, defaultDestination string
	defaultConfigurationForwarding := &forwarding.Configuration{}
	defaultConfigurationSource := &forwarding.Configuration{}
	defaultConfigurationDestination := &forwarding.Configuration{}
	if defaults, ok := configuration.Forwarding["defaults"]; ok {
		defaultSource = defaults.Source
		defaultDestination = defaults.Destination
		defaultConfigurationForwarding = defaults.Configuration.ToInternalConfiguration()
		if err := defaultConfigurationForwarding.EnsureValid(false); err != nil {
			return fmt.Errorf("invalid default forwarding configuration: %w", err)
		}
		defaultConfigurationSource = defaults.ConfigurationSource.ToInternalConfiguration()
		if err := defaultConfigurationSource.EnsureValid(true); err != nil {
			return fmt.Errorf("invalid default forwarding source configuration: %w", err)
		}
		defaultConfigurationDestination = defaults.ConfigurationDestination.ToInternalConfiguration()
		if err := defaultConfigurationDestination.EnsureValid(true); err != nil {
			return fmt.Errorf("invalid default forwarding destination configuration: %w", err)
		}
	}

	// Extract and validate synchronization defaults.
	var defaultAlpha, defaultBeta string
	var defaultFlushOnCreate project.FlushOnCreateBehavior
	defaultConfigurationSynchronization := &synchronization.Configuration{}
	defaultConfigurationAlpha := &synchronization.Configuration{}
	defaultConfigurationBeta := &synchronization.Configuration{}
	if defaults, ok := configuration.Synchronization["defaults"]; ok {
		defaultAlpha = defaults.Alpha
		defaultBeta = defaults.Beta
		defaultFlushOnCreate = defaults.FlushOnCreate
		defaultConfigurationSynchronization = defaults.Configuration.ToInternalConfiguration()
		if err := defaultConfigurationSynchronization.EnsureValid(false); err != nil {
			return fmt.Errorf("invalid default synchronization configuration: %w", err)
		}
		defaultConfigurationAlpha = defaults.ConfigurationAlpha.ToInternalConfiguration()
		if err := defaultConfigurationAlpha.EnsureValid(true); err != nil {
			return fmt.Errorf("invalid default synchronization alpha configuration: %w", err)
		}
		defaultConfigurationBeta = defaults.ConfigurationBeta.ToInternalConfiguration()
		if err := defaultConfigurationBeta.EnsureValid(true); err != nil {
			return fmt.Errorf("invalid default synchronization beta configuration: %w", err)
		}
	}

	// Merge global and default configurations, with defaults taking priority.
	defaultConfigurationForwarding = forwarding.MergeConfigurations(
		globalConfigurationForwarding,
		defaultConfigurationForwarding,
	)
	defaultConfigurationSynchronization = synchronization.MergeConfigurations(
		globalConfigurationSynchronization,
		defaultConfigurationSynchronization,
	)

	// Generate forward session creation specifications.
	var forwardingSpecifications []*forwardingsvc.CreationSpecification
	for name, session := range configuration.Forwarding {
		// Ignore defaults.
		if name == "defaults" {
			continue
		}

		// Verify that the name is valid.
		if err := selection.EnsureNameValid(name); err != nil {
			return fmt.Errorf("invalid forwarding session name (%s): %v", name, err)
		}

		// Compute URLs.
		source := session.Source
		if source == "" {
			source = defaultSource
		}
		destination := session.Destination
		if destination == "" {
			destination = defaultDestination
		}

		// Parse URLs.
		sourceURL, err := url.Parse(source, url.Kind_Forwarding, true)
		if err != nil {
			return fmt.Errorf("unable to parse forwarding source URL (%s): %v", source, err)
		}
		destinationURL, err := url.Parse(destination, url.Kind_Forwarding, false)
		if err != nil {
			return fmt.Errorf("unable to parse forwarding destination URL (%s): %v", destination, err)
		}

		// Compute configuration.
		configuration := session.Configuration.ToInternalConfiguration()
		if err := configuration.EnsureValid(false); err != nil {
			return fmt.Errorf("invalid forwarding session configuration for %s: %v", name, err)
		}
		configuration = forwarding.MergeConfigurations(defaultConfigurationForwarding, configuration)

		// Compute source-specific configuration.
		sourceConfiguration := session.ConfigurationSource.ToInternalConfiguration()
		if err := sourceConfiguration.EnsureValid(true); err != nil {
			return fmt.Errorf("invalid forwarding session source configuration for %s: %v", name, err)
		}
		sourceConfiguration = forwarding.MergeConfigurations(defaultConfigurationSource, sourceConfiguration)

		// Compute destination-specific configuration.
		destinationConfiguration := session.ConfigurationDestination.ToInternalConfiguration()
		if err := destinationConfiguration.EnsureValid(true); err != nil {
			return fmt.Errorf("invalid forwarding session destination configuration for %s: %v", name, err)
		}
		destinationConfiguration = forwarding.MergeConfigurations(defaultConfigurationDestination, destinationConfiguration)

		// Record the specification.
		forwardingSpecifications = append(forwardingSpecifications, &forwardingsvc.CreationSpecification{
			Source:                   sourceURL,
			Destination:              destinationURL,
			Configuration:            configuration,
			ConfigurationSource:      sourceConfiguration,
			ConfigurationDestination: destinationConfiguration,
			Name:                     name,
			Labels: map[string]string{
				project.LabelKey: identifier,
			},
			Paused: startConfiguration.paused,
		})
	}

	// Generate synchronization session creation specifications and keep track
	// of those that we should flush on creation.
	var synchronizationSpecifications []*synchronizationsvc.CreationSpecification
	var flushOnCreateByIndex []bool
	for name, session := range configuration.Synchronization {
		// Ignore defaults.
		if name == "defaults" {
			continue
		}

		// Verify that the name is valid.
		if err := selection.EnsureNameValid(name); err != nil {
			return fmt.Errorf("invalid synchronization session name (%s): %v", name, err)
		}

		// Compute URLs.
		alpha := session.Alpha
		if alpha == "" {
			alpha = defaultAlpha
		}
		beta := session.Beta
		if beta == "" {
			beta = defaultBeta
		}

		// Parse URLs.
		alphaURL, err := url.Parse(alpha, url.Kind_Synchronization, true)
		if err != nil {
			return fmt.Errorf("unable to parse synchronization alpha URL (%s): %v", alpha, err)
		}
		betaURL, err := url.Parse(beta, url.Kind_Synchronization, false)
		if err != nil {
			return fmt.Errorf("unable to parse synchronization beta URL (%s): %v", beta, err)
		}

		// Compute configuration.
		configuration := session.Configuration.ToInternalConfiguration()
		if err := configuration.EnsureValid(false); err != nil {
			return fmt.Errorf("invalid synchronization session configuration for %s: %v", name, err)
		}
		configuration = synchronization.MergeConfigurations(defaultConfigurationSynchronization, configuration)

		// Compute alpha-specific configuration.
		alphaConfiguration := session.ConfigurationAlpha.ToInternalConfiguration()
		if err := alphaConfiguration.EnsureValid(true); err != nil {
			return fmt.Errorf("invalid synchronization session alpha configuration for %s: %v", name, err)
		}
		alphaConfiguration = synchronization.MergeConfigurations(defaultConfigurationAlpha, alphaConfiguration)

		// Compute beta-specific configuration.
		betaConfiguration := session.ConfigurationBeta.ToInternalConfiguration()
		if err := betaConfiguration.EnsureValid(true); err != nil {
			return fmt.Errorf("invalid synchronization session beta configuration for %s: %v", name, err)
		}
		betaConfiguration = synchronization.MergeConfigurations(defaultConfigurationBeta, betaConfiguration)

		// Record the specification.
		synchronizationSpecifications = append(synchronizationSpecifications, &synchronizationsvc.CreationSpecification{
			Alpha:              alphaURL,
			Beta:               betaURL,
			Configuration:      configuration,
			ConfigurationAlpha: alphaConfiguration,
			ConfigurationBeta:  betaConfiguration,
			Name:               name,
			Labels: map[string]string{
				project.LabelKey: identifier,
			},
			Paused: startConfiguration.paused,
		})

		// Compute and store flush-on-creation behavior.
		if session.FlushOnCreate.IsDefault() {
			flushOnCreateByIndex = append(flushOnCreateByIndex, defaultFlushOnCreate.FlushOnCreate())
		} else {
			flushOnCreateByIndex = append(flushOnCreateByIndex, session.FlushOnCreate.FlushOnCreate())
		}
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.Connect(true, true)
	if err != nil {
		return fmt.Errorf("unable to connect to daemon: %w", err)
	}
	defer daemonConnection.Close()

	// At this point, we're going to try to create resources, so we need to
	// maintain the lock file in case even some of them are successful.
	removeLockFileOnReturn = false

	// Perform pre-creation commands.
	for _, command := range configuration.BeforeCreate {
		fmt.Println(">", command)
		if err := runInShell(command); err != nil {
			return fmt.Errorf("pre-create command failed: %w", err)
		}
	}

	// Create forwarding sessions.
	for _, specification := range forwardingSpecifications {
		if _, err := forward.CreateWithSpecification(daemonConnection, specification); err != nil {
			return fmt.Errorf("unable to create forwarding session (%s): %v", specification.Name, err)
		}
	}

	// Create synchronization sessions and track those that we should flush.
	var sessionsToFlush []string
	for s, specification := range synchronizationSpecifications {
		// Perform session creation.
		session, err := sync.CreateWithSpecification(daemonConnection, specification)
		if err != nil {
			return fmt.Errorf("unable to create synchronization session (%s): %v", specification.Name, err)
		}

		// Determine whether or not to flush this session.
		if !startConfiguration.paused && flushOnCreateByIndex[s] {
			sessionsToFlush = append(sessionsToFlush, session)
		}
	}

	// Flush synchronization sessions for which flushing has been requested.
	if len(sessionsToFlush) > 0 {
		flushSelection := &selection.Selection{Specifications: sessionsToFlush}
		if err := sync.FlushWithSelection(daemonConnection, flushSelection, false); err != nil {
			return fmt.Errorf("unable to flush synchronization session(s): %w", err)
		}
	}

	// Perform post-creation commands.
	for _, command := range configuration.AfterCreate {
		fmt.Println(">", command)
		if err := runInShell(command); err != nil {
			return fmt.Errorf("post-create command failed: %w", err)
		}
	}

	// Success.
	return nil
}

// startCommand is the start command.
var startCommand = &cobra.Command{
	Use:          "start",
	Short:        "Start project sessions",
	Args:         cmd.DisallowArguments,
	RunE:         startMain,
	SilenceUsage: true,
}

// startConfiguration stores configuration for the start command.
var startConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
	// projectFile is the path to the project file, if non-default.
	projectFile string
	// paused indicates whether or not to create sessions in a pre-paused state.
	paused bool
	// noGlobalConfiguration specifies whether or not the global configuration
	// file should be ignored.
	noGlobalConfiguration bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := startCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&startConfiguration.help, "help", "h", false, "Show help information")

	// Wire up project file flags.
	flags.StringVarP(&startConfiguration.projectFile, "project-file", "f", "", "Specify project file")

	// Wire up paused flags.
	flags.BoolVarP(&startConfiguration.paused, "paused", "p", false, "Create the session pre-paused")

	// Wire up general configuration flags.
	flags.BoolVar(&startConfiguration.noGlobalConfiguration, "no-global-configuration", false, "Ignore the global configuration file")
}
