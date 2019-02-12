package main

import (
	"fmt"

	"github.com/dustin/go-humanize"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	sessionpkg "github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/sync"
)

func printSession(state *sessionpkg.State, long bool) {
	// Print the session identifier.
	fmt.Println("Session:", state.Session.Identifier)

	// Print extended information, if desired.
	if long {
		// Extract configuration.
		configuration := state.Session.Configuration

		// Compute and print synchronization mode.
		synchronizationMode := configuration.SynchronizationMode.Description()
		if configuration.SynchronizationMode.IsDefault() {
			defaultSynchronizationMode := state.Session.Version.DefaultSynchronizationMode()
			synchronizationMode += fmt.Sprintf(" (%s)", defaultSynchronizationMode.Description())
		}
		fmt.Println("Synchronization mode:", synchronizationMode)

		// Compute and print maximum entry count.
		if configuration.MaximumEntryCount == 0 {
			fmt.Println("Maximum entry count: Unlimited")
		} else {
			fmt.Println("Maximum entry count:", configuration.MaximumEntryCount)
		}

		// Compute and print maximum staging file size.
		if configuration.MaximumStagingFileSize == 0 {
			fmt.Println("Maximum staging file size: Unlimited")
		} else {
			fmt.Printf(
				"Maximum staging file size: %d (%s)\n",
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
		fmt.Println("Symbolic link mode:", symlinkModeDescription)

		// Compute and print the watch mode.
		watchModeDescription := configuration.WatchMode.Description()
		if configuration.WatchMode.IsDefault() {
			defaultWatchMode := state.Session.Version.DefaultWatchMode()
			watchModeDescription += fmt.Sprintf(" (%s)", defaultWatchMode.Description())
		}
		fmt.Println("Watch mode:", watchModeDescription)

		// Compute and print the watch polling interval.
		var watchPollingIntervalDescription string
		if configuration.WatchPollingInterval == 0 {
			watchPollingIntervalDescription = fmt.Sprintf("Default (%d seconds)", filesystem.DefaultPollingInterval)
		} else {
			watchPollingIntervalDescription = fmt.Sprintf("%d seconds", configuration.WatchPollingInterval)
		}
		fmt.Println("Watch polling interval:", watchPollingIntervalDescription)

		// Compute and print the VCS ignore mode.
		ignoreVCSModeDescription := configuration.IgnoreVCSMode.Description()
		if configuration.IgnoreVCSMode == sync.IgnoreVCSMode_IgnoreVCSDefault {
			defaultIgnoreVCSMode := state.Session.Version.DefaultIgnoreVCSMode()
			ignoreVCSModeDescription += fmt.Sprintf(" (%s)", defaultIgnoreVCSMode.Description())
		}
		fmt.Println("Ignore VCS mode:", ignoreVCSModeDescription)

		// Print default ignores.
		if len(configuration.DefaultIgnores) > 0 {
			fmt.Println("Default ignores:")
			for _, p := range configuration.DefaultIgnores {
				fmt.Printf("\t%s\n", p)
			}
		} else {
			fmt.Println("Default ignores: None")
		}

		// Print per-session ignores.
		if len(configuration.Ignores) > 0 {
			fmt.Println("Ignores:")
			for _, p := range configuration.Ignores {
				fmt.Printf("\t%s\n", p)
			}
		} else {
			fmt.Println("Ignores: None")
		}

		// Compute alpha-specific configuration and print permission settings.
		alphaConfigurationMerged := sessionpkg.MergeConfigurations(
			state.Session.Configuration,
			state.Session.ConfigurationAlpha,
		)
		alphaDefaultFileMode := filesystem.Mode(alphaConfigurationMerged.DefaultFileMode)
		alphaDefaultDirectoryMode := filesystem.Mode(alphaConfigurationMerged.DefaultDirectoryMode)
		alphaDefaultOwner := alphaConfigurationMerged.DefaultOwner
		alphaDefaultGroup := alphaConfigurationMerged.DefaultGroup
		alphaPermissionsNonDefault := alphaDefaultFileMode != 0 ||
			alphaDefaultDirectoryMode != 0 ||
			alphaDefaultOwner != "" ||
			alphaDefaultGroup != ""
		if alphaPermissionsNonDefault {
			fmt.Println("Alpha permissions (non-defaults):")
			if alphaDefaultFileMode != 0 {
				fmt.Printf("\tFile mode: %#o\n", alphaDefaultFileMode)
			}
			if alphaDefaultDirectoryMode != 0 {
				fmt.Printf("\tDirectory mode: %#o\n", alphaDefaultDirectoryMode)
			}
			if alphaDefaultOwner != "" {
				fmt.Println("\tOwner:", alphaDefaultOwner)
			}
			if alphaDefaultGroup != "" {
				fmt.Println("\tGroup:", alphaDefaultGroup)
			}
		} else {
			fmt.Println("Alpha permissions: Default")
		}

		// Compute beta-specific configuration and print permission settings.
		betaConfigurationMerged := sessionpkg.MergeConfigurations(
			state.Session.Configuration,
			state.Session.ConfigurationBeta,
		)
		betaDefaultFileMode := filesystem.Mode(betaConfigurationMerged.DefaultFileMode)
		betaDefaultDirectoryMode := filesystem.Mode(betaConfigurationMerged.DefaultDirectoryMode)
		betaDefaultOwner := betaConfigurationMerged.DefaultOwner
		betaDefaultGroup := betaConfigurationMerged.DefaultGroup
		betaPermissionsNonDefault := betaDefaultFileMode != 0 ||
			betaDefaultDirectoryMode != 0 ||
			betaDefaultOwner != "" ||
			betaDefaultGroup != ""
		if betaPermissionsNonDefault {
			fmt.Println("Beta permissions (non-defaults):")
			if betaDefaultFileMode != 0 {
				fmt.Printf("\tFile mode: %#o\n", betaDefaultFileMode)
			}
			if betaDefaultDirectoryMode != 0 {
				fmt.Printf("\tDirectory mode: %#o\n", betaDefaultDirectoryMode)
			}
			if betaDefaultOwner != "" {
				fmt.Println("\tOwner:", betaDefaultOwner)
			}
			if betaDefaultGroup != "" {
				fmt.Println("\tGroup:", betaDefaultGroup)
			}
		} else {
			fmt.Println("Beta permissions: Default")
		}
	}
}
