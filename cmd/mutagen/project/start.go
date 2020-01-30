package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"
	"github.com/mutagen-io/mutagen/cmd/mutagen/forward"
	"github.com/mutagen-io/mutagen/cmd/mutagen/sync"
	"github.com/mutagen-io/mutagen/pkg/configuration/global"
	"github.com/mutagen-io/mutagen/pkg/configuration/legacy"
	projectcfg "github.com/mutagen-io/mutagen/pkg/configuration/project"
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

func startMain(command *cobra.Command, arguments []string) error {
	// Compute the name of the configuration file and change our working
	// directory to the path in which the file resides.
	var configurationFileName string
	if len(arguments) == 0 {
		configurationFileName = project.DefaultConfigurationFileName
	} else if len(arguments) == 1 {
		// Parse the target into directory and file name.
		var directory string
		directory, configurationFileName = filepath.Split(arguments[0])
		if configurationFileName == "" {
			return errors.New("empty configuration file name")
		}

		// Switch to the directory (if it's not the current directory). This is
		// required for relative paths (including relative synchronization paths
		// and relative Unix Domain Socket paths) to be resolved relative to the
		// project configuration file.
		if directory != "" {
			if err := os.Chdir(directory); err != nil {
				return errors.Wrap(err, "unable to switch to target directory")
			}
		}
	} else {
		return errors.New("invalid number of arguments")
	}

	// Compute the lock path.
	lockPath := configurationFileName + project.LockFileExtension

	// Track whether or not we should remove the lock file on return.
	var removeLockFileOnReturn bool

	// Create a locker and defer its closure and potential removal. On Windows
	// systems, we have to handle this removal after the file is closed.
	locker, err := locking.NewLocker(lockPath, 0600)
	if err != nil {
		return errors.Wrap(err, "unable to create project locker")
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
		return errors.Wrap(err, "unable to acquire project lock")
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
		return errors.Wrap(err, "unable to read project lock")
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
		return errors.Wrap(err, "unable to generate project identifier")
	}

	// Write the project identifier to the lock file.
	if _, err := locker.Write([]byte(identifier)); err != nil {
		return errors.Wrap(err, "unable to write project identifier")
	}

	// Load the configuration file.
	configuration, err := projectcfg.LoadConfiguration(configurationFileName)
	if err != nil {
		return errors.Wrap(err, "unable to load configuration file")
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
			return errors.Wrap(err, "unable to compute path to global configuration file")
		}

		// Load the configuration. We allow it do not exist, but we don't fall
		// back to legacy configuration options.
		globalConfiguration, err := global.LoadConfiguration(globalConfigurationPath)
		if err != nil {
			if os.IsNotExist(err) {
				// Compute the path to the global configuration file.
				legacyGlobalConfigurationPath, err := legacy.ConfigurationPath()
				if err != nil {
					return errors.Wrap(err, "unable to compute path to legacy global configuration file")
				}

				// Error out if it exists, we don't fall back to it.
				if _, err := os.Stat(legacyGlobalConfigurationPath); err == nil {
					return errors.Wrap(err, "project infrastructure doesn't support legacy global TOML configuration")
				}
			} else {
				return errors.Wrap(err, "unable to load global configuration")
			}
		} else {
			globalConfigurationForwarding = globalConfiguration.Forwarding.Defaults.Configuration()
			if err := globalConfigurationForwarding.EnsureValid(false); err != nil {
				return errors.Wrap(err, "invalid global forwarding configuration")
			}
			globalConfigurationSynchronization = globalConfiguration.Synchronization.Defaults.Configuration()
			if err := globalConfigurationSynchronization.EnsureValid(false); err != nil {
				return errors.Wrap(err, "invalid global synchronization configuration")
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
		defaultConfigurationForwarding = defaults.Configuration.Configuration()
		if err := defaultConfigurationForwarding.EnsureValid(false); err != nil {
			return errors.Wrap(err, "invalid default forwarding configuration")
		}
		defaultConfigurationSource = defaults.ConfigurationSource.Configuration()
		if err := defaultConfigurationSource.EnsureValid(true); err != nil {
			return errors.Wrap(err, "invalid default forwarding source configuration")
		}
		defaultConfigurationDestination = defaults.ConfigurationDestination.Configuration()
		if err := defaultConfigurationDestination.EnsureValid(true); err != nil {
			return errors.Wrap(err, "invalid default forwarding destination configuration")
		}
	}

	// Extract and validate synchronization defaults.
	var defaultAlpha, defaultBeta string
	defaultConfigurationSynchronization := &synchronization.Configuration{}
	defaultConfigurationAlpha := &synchronization.Configuration{}
	defaultConfigurationBeta := &synchronization.Configuration{}
	if defaults, ok := configuration.Synchronization["defaults"]; ok {
		defaultAlpha = defaults.Alpha
		defaultBeta = defaults.Beta
		defaultConfigurationSynchronization = defaults.Configuration.Configuration()
		if err := defaultConfigurationSynchronization.EnsureValid(false); err != nil {
			return errors.Wrap(err, "invalid default synchronization configuration")
		}
		defaultConfigurationAlpha = defaults.ConfigurationAlpha.Configuration()
		if err := defaultConfigurationAlpha.EnsureValid(true); err != nil {
			return errors.Wrap(err, "invalid default synchronization alpha configuration")
		}
		defaultConfigurationBeta = defaults.ConfigurationBeta.Configuration()
		if err := defaultConfigurationBeta.EnsureValid(true); err != nil {
			return errors.Wrap(err, "invalid default synchronization beta configuration")
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
			return errors.Errorf("invalid forwarding session name (%s): %v", name, err)
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
			return errors.Errorf("unable to parse forwarding source URL (%s): %v", source, err)
		}
		destinationURL, err := url.Parse(destination, url.Kind_Forwarding, false)
		if err != nil {
			return errors.Errorf("unable to parse forwarding destination URL (%s): %v", destination, err)
		}

		// Compute configuration.
		configuration := session.Configuration.Configuration()
		if err := configuration.EnsureValid(false); err != nil {
			return errors.Errorf("invalid forwarding session configuration for %s: %v", name, err)
		}
		configuration = forwarding.MergeConfigurations(defaultConfigurationForwarding, configuration)

		// Compute source-specific configuration.
		sourceConfiguration := session.ConfigurationSource.Configuration()
		if err := sourceConfiguration.EnsureValid(true); err != nil {
			return errors.Errorf("invalid forwarding session source configuration for %s: %v", name, err)
		}
		sourceConfiguration = forwarding.MergeConfigurations(defaultConfigurationSource, sourceConfiguration)

		// Compute destination-specific configuration.
		destinationConfiguration := session.ConfigurationDestination.Configuration()
		if err := destinationConfiguration.EnsureValid(true); err != nil {
			return errors.Errorf("invalid forwarding session destination configuration for %s: %v", name, err)
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

	// Generate synchronization session creation specifications.
	var synchronizationSpecifications []*synchronizationsvc.CreationSpecification
	for name, session := range configuration.Synchronization {
		// Ignore defaults.
		if name == "defaults" {
			continue
		}

		// Verify that the name is valid.
		if err := selection.EnsureNameValid(name); err != nil {
			return errors.Errorf("invalid synchronization session name (%s): %v", name, err)
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
			return errors.Errorf("unable to parse synchronization alpha URL (%s): %v", alpha, err)
		}
		betaURL, err := url.Parse(beta, url.Kind_Synchronization, false)
		if err != nil {
			return errors.Errorf("unable to parse synchronization beta URL (%s): %v", beta, err)
		}

		// Compute configuration.
		configuration := session.Configuration.Configuration()
		if err := configuration.EnsureValid(false); err != nil {
			return errors.Errorf("invalid synchronization session configuration for %s: %v", name, err)
		}
		configuration = synchronization.MergeConfigurations(defaultConfigurationSynchronization, configuration)

		// Compute alpha-specific configuration.
		alphaConfiguration := session.ConfigurationAlpha.Configuration()
		if err := alphaConfiguration.EnsureValid(true); err != nil {
			return errors.Errorf("invalid synchronization session alpha configuration for %s: %v", name, err)
		}
		alphaConfiguration = synchronization.MergeConfigurations(defaultConfigurationAlpha, alphaConfiguration)

		// Compute beta-specific configuration.
		betaConfiguration := session.ConfigurationBeta.Configuration()
		if err := betaConfiguration.EnsureValid(true); err != nil {
			return errors.Errorf("invalid synchronization session beta configuration for %s: %v", name, err)
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
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.CreateClientConnection(true, true)
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// At this point, we're going to try to create resources, so we need to
	// maintain the lock file in case even some of them are successful.
	removeLockFileOnReturn = false

	// Perform setup commands.
	for _, command := range configuration.Setup {
		fmt.Println(">", command)
		if err := runCommand(command); err != nil {
			return errors.Wrap(err, "setup command failed")
		}
	}

	// Create forwarding sessions.
	forwardingService := forwardingsvc.NewForwardingClient(daemonConnection)
	for _, specification := range forwardingSpecifications {
		if err := forward.CreateWithSpecification(forwardingService, specification); err != nil {
			return errors.Errorf("unable to create forwarding session (%s): %v", specification.Name, err)
		}
	}

	// Create synchronization sessions.
	synchronizationService := synchronizationsvc.NewSynchronizationClient(daemonConnection)
	for _, specification := range synchronizationSpecifications {
		if err := sync.CreateWithSpecification(synchronizationService, specification); err != nil {
			return errors.Errorf("unable to create synchronization session (%s): %v", specification.Name, err)
		}
	}

	// Success.
	return nil
}

var startCommand = &cobra.Command{
	Use:          "start [<configuration-file>]",
	Short:        "Start project sessions",
	RunE:         startMain,
	SilenceUsage: true,
}

var startConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
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

	// Wire up paused flags.
	flags.BoolVarP(&startConfiguration.paused, "paused", "p", false, "Create the session pre-paused")

	// Wire up general configuration flags.
	flags.BoolVar(&startConfiguration.noGlobalConfiguration, "no-global-configuration", false, "Ignore the global configuration file")
}
