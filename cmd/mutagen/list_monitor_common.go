package main

import (
	"fmt"
	"math"

	"github.com/dustin/go-humanize"

	sessionpkg "github.com/havoc-io/mutagen/pkg/session"
	urlpkg "github.com/havoc-io/mutagen/pkg/url"
)

const (
	// maxUint64Description is a human-friendly mathematic description of
	// math.MaxUint64.
	maxUint64Description = "2⁶⁴−1"

	// emptyLabelValueDescription is a human-friendly description representing
	// an empty label value. It contains characters which are invalid for use in
	// label values, so it won't be confused for one.
	emptyLabelValueDescription = "<empty>"
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
	if configuration.WatchMode != sessionpkg.WatchMode_WatchModeNoWatch {
		var watchPollingIntervalDescription string
		if configuration.WatchPollingInterval == 0 {
			watchPollingIntervalDescription = fmt.Sprintf("Default (%d seconds)", version.DefaultWatchPollingInterval())
		} else {
			watchPollingIntervalDescription = fmt.Sprintf("%d seconds", configuration.WatchPollingInterval)
		}
		fmt.Println("\tWatch polling interval:", watchPollingIntervalDescription)
	}

	// Compute and print the probe mode.
	probeModeDescription := configuration.ProbeMode.Description()
	if configuration.ProbeMode.IsDefault() {
		probeModeDescription += fmt.Sprintf(" (%s)", version.DefaultProbeMode().Description())
	}
	fmt.Println("\tProbe mode:", probeModeDescription)

	// Compute and print the scan mode.
	scanModeDescription := configuration.ScanMode.Description()
	if configuration.ScanMode.IsDefault() {
		scanModeDescription += fmt.Sprintf(" (%s)", version.DefaultScanMode().Description())
	}
	fmt.Println("\tScan mode:", scanModeDescription)

	// Compute and print the staging mode.
	stageModeDescription := configuration.StageMode.Description()
	if configuration.StageMode.IsDefault() {
		stageModeDescription += fmt.Sprintf(" (%s)", version.DefaultStageMode().Description())
	}
	fmt.Println("\tStage mode:", stageModeDescription)

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

	// Print labels.
	if len(state.Session.Labels) > 0 {
		fmt.Println("Labels:")
		keys := sessionpkg.ExtractAndSortLabelKeys(state.Session.Labels)
		for _, key := range keys {
			value := state.Session.Labels[key]
			if value == "" {
				value = emptyLabelValueDescription
			}
			fmt.Printf("\t%s: %s\n", key, value)
		}
	} else {
		fmt.Println("Labels: None")
	}

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
		var maximumEntryCountDescription string
		if configuration.MaximumEntryCount == 0 {
			if m := state.Session.Version.DefaultMaximumEntryCount(); m == math.MaxUint64 {
				maximumEntryCountDescription = fmt.Sprintf("Default (%s)", maxUint64Description)
			} else {
				maximumEntryCountDescription = fmt.Sprintf("Default (%d)", m)
			}
		} else {
			maximumEntryCountDescription = fmt.Sprintf("%d", configuration.MaximumEntryCount)
		}
		fmt.Println("\tMaximum allowed entry count:", maximumEntryCountDescription)

		// Compute and print maximum staging file size.
		var maximumStagingFileSizeDescription string
		if configuration.MaximumStagingFileSize == 0 {
			maximumStagingFileSizeDescription = fmt.Sprintf(
				"Default (%s)",
				humanize.Bytes(state.Session.Version.DefaultMaximumStagingFileSize()),
			)
		} else {
			maximumStagingFileSizeDescription = fmt.Sprintf(
				"%d (%s)",
				configuration.MaximumStagingFileSize,
				humanize.Bytes(configuration.MaximumStagingFileSize),
			)
		}
		fmt.Println("\tMaximum staging file size:", maximumStagingFileSizeDescription)

		// Compute and print symlink mode.
		symlinkModeDescription := configuration.SymlinkMode.Description()
		if configuration.SymlinkMode.IsDefault() {
			defaultSymlinkMode := state.Session.Version.DefaultSymlinkMode()
			symlinkModeDescription += fmt.Sprintf(" (%s)", defaultSymlinkMode.Description())
		}
		fmt.Println("\tSymbolic link mode:", symlinkModeDescription)

		// Compute and print the VCS ignore mode.
		ignoreVCSModeDescription := configuration.IgnoreVCSMode.Description()
		if configuration.IgnoreVCSMode.IsDefault() {
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
