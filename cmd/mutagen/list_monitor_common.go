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
		// Print default and per-session ignores.
		if len(state.Session.GlobalConfiguration.Ignores) > 0 {
			fmt.Println("Default ignores:")
			for _, p := range state.Session.GlobalConfiguration.Ignores {
				fmt.Printf("\t%s\n", p)
			}
		}
		if len(state.Session.Configuration.Ignores) > 0 {
			fmt.Println("Ignores:")
			for _, p := range state.Session.Configuration.Ignores {
				fmt.Printf("\t%s\n", p)
			}
		}

		// Compute the merged session configuration.
		mergedConfiguration := sessionpkg.MergeConfigurations(
			state.Session.Configuration,
			state.Session.GlobalConfiguration,
		)

		// Compute and print symlink mode.
		symlinkModeDescription := mergedConfiguration.SymlinkMode.Description()
		if mergedConfiguration.SymlinkMode == sync.SymlinkMode_Default {
			defaultSymlinkMode := state.Session.Version.DefaultSymlinkMode()
			symlinkModeDescription += fmt.Sprintf(" (%s)", defaultSymlinkMode.Description())
		}
		fmt.Println("Symlink mode:", symlinkModeDescription)

		// Compute and print the watch mode.
		watchModeDescription := mergedConfiguration.WatchMode.Description()
		if mergedConfiguration.WatchMode == filesystem.WatchMode_Default {
			defaultWatchMode := state.Session.Version.DefaultWatchMode()
			watchModeDescription += fmt.Sprintf(" (%s)", defaultWatchMode.Description())
		}
		fmt.Println("Watch mode:", watchModeDescription)

		// Compute and print the polling interval.
		var watchPollingIntervalDescription string
		if mergedConfiguration.WatchPollingInterval == 0 {
			watchPollingIntervalDescription = fmt.Sprintf("Default (%d seconds)", filesystem.DefaultPollingInterval)
		} else {
			watchPollingIntervalDescription = fmt.Sprintf("%d seconds", mergedConfiguration.WatchPollingInterval)
		}
		fmt.Println("Watch polling interval:", watchPollingIntervalDescription)
	}
}
