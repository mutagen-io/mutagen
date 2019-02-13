package main

import (
	"fmt"

	"github.com/dustin/go-humanize"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	sessionpkg "github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/sync"
	urlpkg "github.com/havoc-io/mutagen/pkg/url"
)

func printEndpoint(name string, url *urlpkg.URL, configuration *sessionpkg.Configuration, version sessionpkg.Version) {
	// Print the endpoint header.
	fmt.Println(name, "configuration:")

	// Print the URL.
	fmt.Println("\tURL:", url.Format("\n\t\t"))

	// Compute and print the watch mode.
	watchModeDescription := configuration.WatchMode.Description()
	if configuration.WatchMode.IsDefault() {
		watchModeDescription += fmt.Sprintf(" (%s)", version.DefaultWatchMode().Description())
	}
	fmt.Println("\tWatch mode:", watchModeDescription)

	// Compute and print the watch polling interval, so long as we're not in
	// no-watch mode.
	if configuration.WatchMode != filesystem.WatchMode_WatchModeNoWatch {
		var watchPollingIntervalDescription string
		if configuration.WatchPollingInterval == 0 {
			watchPollingIntervalDescription = fmt.Sprintf("Default (%d seconds)", version.DefaultWatchPollingInterval())
		} else {
			watchPollingIntervalDescription = fmt.Sprintf("%d seconds", configuration.WatchPollingInterval)
		}
		fmt.Println("\tWatch polling interval:", watchPollingIntervalDescription)
	}

	// Compute and print the default file mode.
	var defaultFileModeDescription string
	if configuration.DefaultFileMode == 0 {
		defaultFileModeDescription = fmt.Sprintf("Default (%#o)", version.DefaultFileMode())
	} else {
		defaultFileModeDescription = fmt.Sprintf("%#o", configuration.DefaultFileMode)
	}
	fmt.Println("\tFile mode:", defaultFileModeDescription)

	// Compute and print the default directory mode.
	var defaultDirectoryModeDescription string
	if configuration.DefaultDirectoryMode == 0 {
		defaultDirectoryModeDescription = fmt.Sprintf("Default (%#o)", version.DefaultDirectoryMode())
	} else {
		defaultDirectoryModeDescription = fmt.Sprintf("%#o", configuration.DefaultDirectoryMode)
	}
	fmt.Println("\tDirectory mode:", defaultDirectoryModeDescription)

	// Compute and print the default file/directory owner.
	defaultOwnerDescription := "Default"
	if configuration.DefaultOwner != "" {
		defaultOwnerDescription = configuration.DefaultOwner
	}
	fmt.Println("\tDefault file/directory owner:", defaultOwnerDescription)

	// Compute and print the default file/directory group.
	defaultGroupDescription := "Default"
	if configuration.DefaultGroup != "" {
		defaultGroupDescription = configuration.DefaultGroup
	}
	fmt.Println("\tDefault file/directory group:", defaultGroupDescription)
}

func printSession(state *sessionpkg.State, long bool) {
	// Print the session identifier.
	fmt.Println("Session:", state.Session.Identifier)

	// Print extended information, if desired.
	if long {
		// Print the configuration header.
		fmt.Println("Configuration:")

		// Extract configuration.
		configuration := state.Session.Configuration

		// Compute and print synchronization mode.
		synchronizationMode := configuration.SynchronizationMode.Description()
		if configuration.SynchronizationMode.IsDefault() {
			defaultSynchronizationMode := state.Session.Version.DefaultSynchronizationMode()
			synchronizationMode += fmt.Sprintf(" (%s)", defaultSynchronizationMode.Description())
		}
		fmt.Println("\tSynchronization mode:", synchronizationMode)

		// Compute and print maximum entry count.
		if configuration.MaximumEntryCount == 0 {
			fmt.Println("\tMaximum entry count: Unlimited")
		} else {
			fmt.Println("\tMaximum entry count:", configuration.MaximumEntryCount)
		}

		// Compute and print maximum staging file size.
		if configuration.MaximumStagingFileSize == 0 {
			fmt.Println("\tMaximum staging file size: Unlimited")
		} else {
			fmt.Printf(
				"\tMaximum staging file size: %d (%s)\n",
				configuration.MaximumStagingFileSize,
				humanize.Bytes(configuration.MaximumStagingFileSize),
			)
		}

		// Compute and print symlink mode.
		symlinkModeDescription := configuration.SymlinkMode.Description()
		if configuration.SymlinkMode == sync.SymlinkMode_SymlinkDefault {
			defaultSymlinkMode := state.Session.Version.DefaultSymlinkMode()
			symlinkModeDescription += fmt.Sprintf(" (%s)", defaultSymlinkMode.Description())
		}
		fmt.Println("\tSymbolic link mode:", symlinkModeDescription)

		// Compute and print the VCS ignore mode.
		ignoreVCSModeDescription := configuration.IgnoreVCSMode.Description()
		if configuration.IgnoreVCSMode == sync.IgnoreVCSMode_IgnoreVCSDefault {
			defaultIgnoreVCSMode := state.Session.Version.DefaultIgnoreVCSMode()
			ignoreVCSModeDescription += fmt.Sprintf(" (%s)", defaultIgnoreVCSMode.Description())
		}
		fmt.Println("\tIgnore VCS mode:", ignoreVCSModeDescription)

		// Print default ignores. Since this field is deprecated, we don't print
		// it if it's not set.
		if len(configuration.DefaultIgnores) > 0 {
			fmt.Println("\tDefault ignores:")
			for _, p := range configuration.DefaultIgnores {
				fmt.Printf("\t\t%s\n", p)
			}
		}

		// Print per-session ignores.
		if len(configuration.Ignores) > 0 {
			fmt.Println("\tIgnores:")
			for _, p := range configuration.Ignores {
				fmt.Printf("\t\t%s\n", p)
			}
		} else {
			fmt.Println("\tIgnores: None")
		}

		// Compute and print alpha-specific configuration.
		alphaConfigurationMerged := sessionpkg.MergeConfigurations(
			state.Session.Configuration,
			state.Session.ConfigurationAlpha,
		)
		printEndpoint("Alpha", state.Session.Alpha, alphaConfigurationMerged, state.Session.Version)

		// Compute and print beta-specific configuration.
		betaConfigurationMerged := sessionpkg.MergeConfigurations(
			state.Session.Configuration,
			state.Session.ConfigurationBeta,
		)
		printEndpoint("Beta", state.Session.Beta, betaConfigurationMerged, state.Session.Version)
	}
}
