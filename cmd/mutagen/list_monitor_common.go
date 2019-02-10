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
		if configuration.WatchMode == filesystem.WatchMode_WatchDefault {
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

		// Print alpha permission settings.
		var alphaDefaultFileMode uint32
		var alphaDefaultDirectoryMode uint32
		var alphaDefaultUser string
		var alphaDefaultGroup string
		if configuration.PermissionDefaultFileModeAlpha != 0 {
			alphaDefaultFileMode = configuration.PermissionDefaultFileModeAlpha
		} else if configuration.PermissionDefaultFileMode != 0 {
			alphaDefaultFileMode = configuration.PermissionDefaultFileMode
		}
		if configuration.PermissionDefaultDirectoryModeAlpha != 0 {
			alphaDefaultDirectoryMode = configuration.PermissionDefaultDirectoryModeAlpha
		} else if configuration.PermissionDefaultDirectoryMode != 0 {
			alphaDefaultDirectoryMode = configuration.PermissionDefaultDirectoryMode
		}
		if configuration.PermissionDefaultUserAlpha != "" {
			alphaDefaultUser = configuration.PermissionDefaultUserAlpha
		} else if configuration.PermissionDefaultUser != "" {
			alphaDefaultUser = configuration.PermissionDefaultUser
		}
		if configuration.PermissionDefaultGroupAlpha != "" {
			alphaDefaultGroup = configuration.PermissionDefaultGroupAlpha
		} else if configuration.PermissionDefaultGroup != "" {
			alphaDefaultGroup = configuration.PermissionDefaultGroup
		}
		alphaPermissionsNonDefault := alphaDefaultFileMode != 0 ||
			alphaDefaultDirectoryMode != 0 ||
			alphaDefaultUser != "" ||
			alphaDefaultGroup != ""
		if alphaPermissionsNonDefault {
			fmt.Println("Alpha permissions (non-defaults):")
			if alphaDefaultFileMode != 0 {
				fmt.Printf("\tFile mode: %#o\n", alphaDefaultFileMode)
			}
			if alphaDefaultDirectoryMode != 0 {
				fmt.Printf("\tDirectory mode: %#o\n", alphaDefaultDirectoryMode)
			}
			if alphaDefaultUser != "" {
				fmt.Println("\tOwner user:", alphaDefaultUser)
			}
			if alphaDefaultGroup != "" {
				fmt.Println("\tOwner group:", alphaDefaultGroup)
			}
		} else {
			fmt.Println("Alpha permissions: Default")
		}

		// Print beta permission settings.
		var betaDefaultFileMode uint32
		var betaDefaultDirectoryMode uint32
		var betaDefaultUser string
		var betaDefaultGroup string
		if configuration.PermissionDefaultFileModeBeta != 0 {
			betaDefaultFileMode = configuration.PermissionDefaultFileModeBeta
		} else if configuration.PermissionDefaultFileMode != 0 {
			betaDefaultFileMode = configuration.PermissionDefaultFileMode
		}
		if configuration.PermissionDefaultDirectoryModeBeta != 0 {
			betaDefaultDirectoryMode = configuration.PermissionDefaultDirectoryModeBeta
		} else if configuration.PermissionDefaultDirectoryMode != 0 {
			betaDefaultDirectoryMode = configuration.PermissionDefaultDirectoryMode
		}
		if configuration.PermissionDefaultUserBeta != "" {
			betaDefaultUser = configuration.PermissionDefaultUserBeta
		} else if configuration.PermissionDefaultUser != "" {
			betaDefaultUser = configuration.PermissionDefaultUser
		}
		if configuration.PermissionDefaultGroupBeta != "" {
			betaDefaultGroup = configuration.PermissionDefaultGroupBeta
		} else if configuration.PermissionDefaultGroup != "" {
			betaDefaultGroup = configuration.PermissionDefaultGroup
		}
		betaPermissionsNonDefault := betaDefaultFileMode != 0 ||
			betaDefaultDirectoryMode != 0 ||
			betaDefaultUser != "" ||
			betaDefaultGroup != ""
		if betaPermissionsNonDefault {
			fmt.Println("Beta permissions (non-defaults):")
			if betaDefaultFileMode != 0 {
				fmt.Printf("\tFile mode: %#o\n", betaDefaultFileMode)
			}
			if betaDefaultDirectoryMode != 0 {
				fmt.Printf("\tDirectory mode: %#o\n", betaDefaultDirectoryMode)
			}
			if betaDefaultUser != "" {
				fmt.Println("\tOwner user:", betaDefaultUser)
			}
			if betaDefaultGroup != "" {
				fmt.Println("\tOwner group:", betaDefaultGroup)
			}
		} else {
			fmt.Println("Beta permissions: Default")
		}
	}
}
