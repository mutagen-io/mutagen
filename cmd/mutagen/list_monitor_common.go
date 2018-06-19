package main

import (
	"fmt"

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
		}

		// Print per-session ignores.
		if len(configuration.Ignores) > 0 {
			fmt.Println("Ignores:")
			for _, p := range configuration.Ignores {
				fmt.Printf("\t%s\n", p)
			}
		}

		// Compute and print symlink mode.
		symlinkModeDescription := configuration.SymlinkMode.Description()
		if configuration.SymlinkMode == sync.SymlinkMode_SymlinkDefault {
			defaultSymlinkMode := state.Session.Version.DefaultSymlinkMode()
			symlinkModeDescription += fmt.Sprintf(" (%s)", defaultSymlinkMode.Description())
		}
		fmt.Println("Symlink mode:", symlinkModeDescription)

		// Compute and print the watch mode.
		watchModeDescription := configuration.WatchMode.Description()
		if configuration.WatchMode == filesystem.WatchMode_WatchDefault {
			defaultWatchMode := state.Session.Version.DefaultWatchMode()
			watchModeDescription += fmt.Sprintf(" (%s)", defaultWatchMode.Description())
		}
		fmt.Println("Watch mode:", watchModeDescription)

		// Compute and print the polling interval.
		var watchPollingIntervalDescription string
		if configuration.WatchPollingInterval == 0 {
			watchPollingIntervalDescription = fmt.Sprintf("Default (%d seconds)", filesystem.DefaultPollingInterval)
		} else {
			watchPollingIntervalDescription = fmt.Sprintf("%d seconds", configuration.WatchPollingInterval)
		}
		fmt.Println("Watch polling interval:", watchPollingIntervalDescription)
	}
}
