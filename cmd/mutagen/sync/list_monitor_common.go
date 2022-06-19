package sync

import (
	"fmt"
	"math"

	"github.com/dustin/go-humanize"

	"github.com/fatih/color"

	"github.com/mutagen-io/mutagen/cmd/mutagen/common"

	"github.com/mutagen-io/mutagen/pkg/selection"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
	"github.com/mutagen-io/mutagen/pkg/synchronization/rsync"
	"github.com/mutagen-io/mutagen/pkg/url"
)

const (
	// maxUint64Description is a human-friendly mathematical description of
	// math.MaxUint64.
	maxUint64Description = "2⁶⁴−1"

	// emptyLabelValueDescription is a human-friendly description representing
	// an empty label value. It contains characters which are invalid for use in
	// label values, so it won't be confused for one.
	emptyLabelValueDescription = "<empty>"
)

// formatDirectoryCount formats a directory count for display.
func formatDirectoryCount(count uint64) string {
	if count == 1 {
		return "1 directory"
	}
	return fmt.Sprintf("%d directories", count)
}

// formatFileCountAndSize formats a file count and total size count for display.
func formatFileCountAndSize(count uint64, totalSize uint64) string {
	if count == 1 {
		return fmt.Sprintf("1 file (%s)", humanize.Bytes(totalSize))
	}
	return fmt.Sprintf("%d files (%s)", count, humanize.Bytes(totalSize))
}

// formatSymbolicLinkCount formats a symbolic link count for display.
func formatSymbolicLinkCount(count uint64) string {
	if count == 1 {
		return "1 symbolic link"
	}
	return fmt.Sprintf("%d symbolic links", count)
}

// formatPath formats a path for display.
func formatPath(path string) string {
	if path == "" {
		return "<root>"
	}
	return path
}

// formatEntry formats an entry for display.
func formatEntry(entry *core.Entry) string {
	if entry == nil {
		return "<non-existent>"
	} else if entry.Kind == core.EntryKind_Directory {
		return "Directory"
	} else if entry.Kind == core.EntryKind_File {
		if entry.Executable {
			return fmt.Sprintf("Executable File (%x)", entry.Digest)
		}
		return fmt.Sprintf("File (%x)", entry.Digest)
	} else if entry.Kind == core.EntryKind_SymbolicLink {
		return fmt.Sprintf("Symbolic Link (%s)", entry.Target)
	} else if entry.Kind == core.EntryKind_Untracked {
		return "Untracked content"
	} else if entry.Kind == core.EntryKind_Problematic {
		return fmt.Sprintf("Problematic content (%s)", entry.Problem)
	}
	return "<unknown>"
}

// printEndpoint prints the configuration for a synchronization endpoint.
func printEndpoint(name string, url *url.URL, configuration *synchronization.Configuration, state *synchronization.EndpointState, version synchronization.Version, mode common.SessionDisplayMode) {
	// Print the endpoint header.
	fmt.Printf("%s:\n", name)

	// Print the URL.
	fmt.Println("\tURL:", url.Format("\n\t\t"))

	// Print configuration information if desired.
	if mode == common.SessionDisplayModeListLong || mode == common.SessionDisplayModeMonitorLong {
		// Print configuration header.
		fmt.Println("\tConfiguration:")

		// Compute and print the watch mode.
		watchModeDescription := configuration.WatchMode.Description()
		if configuration.WatchMode.IsDefault() {
			watchModeDescription += fmt.Sprintf(" (%s)", version.DefaultWatchMode().Description())
		}
		fmt.Println("\t\tWatch mode:", watchModeDescription)

		// Compute and print the watch polling interval, so long as we're not in
		// no-watch mode.
		if configuration.WatchMode != synchronization.WatchMode_WatchModeNoWatch {
			var watchPollingIntervalDescription string
			if configuration.WatchPollingInterval == 0 {
				watchPollingIntervalDescription = fmt.Sprintf("Default (%d seconds)", version.DefaultWatchPollingInterval())
			} else {
				watchPollingIntervalDescription = fmt.Sprintf("%d seconds", configuration.WatchPollingInterval)
			}
			fmt.Println("\t\tWatch polling interval:", watchPollingIntervalDescription)
		}

		// Compute and print the probe mode.
		probeModeDescription := configuration.ProbeMode.Description()
		if configuration.ProbeMode.IsDefault() {
			probeModeDescription += fmt.Sprintf(" (%s)", version.DefaultProbeMode().Description())
		}
		fmt.Println("\t\tProbe mode:", probeModeDescription)

		// Compute and print the scan mode.
		scanModeDescription := configuration.ScanMode.Description()
		if configuration.ScanMode.IsDefault() {
			scanModeDescription += fmt.Sprintf(" (%s)", version.DefaultScanMode().Description())
		}
		fmt.Println("\t\tScan mode:", scanModeDescription)

		// Compute and print the staging mode.
		stageModeDescription := configuration.StageMode.Description()
		if configuration.StageMode.IsDefault() {
			stageModeDescription += fmt.Sprintf(" (%s)", version.DefaultStageMode().Description())
		}
		fmt.Println("\t\tStage mode:", stageModeDescription)

		// Compute and print the default file mode.
		var defaultFileModeDescription string
		if configuration.DefaultFileMode == 0 {
			defaultFileModeDescription = fmt.Sprintf("Default (%#o)", version.DefaultFileMode())
		} else {
			defaultFileModeDescription = fmt.Sprintf("%#o", configuration.DefaultFileMode)
		}
		fmt.Println("\t\tFile mode:", defaultFileModeDescription)

		// Compute and print the default directory mode.
		var defaultDirectoryModeDescription string
		if configuration.DefaultDirectoryMode == 0 {
			defaultDirectoryModeDescription = fmt.Sprintf("Default (%#o)", version.DefaultDirectoryMode())
		} else {
			defaultDirectoryModeDescription = fmt.Sprintf("%#o", configuration.DefaultDirectoryMode)
		}
		fmt.Println("\t\tDirectory mode:", defaultDirectoryModeDescription)

		// Compute and print the default file/directory owner.
		defaultOwnerDescription := "Default"
		if configuration.DefaultOwner != "" {
			defaultOwnerDescription = configuration.DefaultOwner
		}
		fmt.Println("\t\tDefault file/directory owner:", defaultOwnerDescription)

		// Compute and print the default file/directory group.
		defaultGroupDescription := "Default"
		if configuration.DefaultGroup != "" {
			defaultGroupDescription = configuration.DefaultGroup
		}
		fmt.Println("\t\tDefault file/directory group:", defaultGroupDescription)
	}

	// At this point, there's no other status information that will be displayed
	// for non-list modes, so we can save ourselves some checks and return if
	// we're in a monitor mode.
	if mode == common.SessionDisplayModeMonitor || mode == common.SessionDisplayModeMonitorLong {
		return
	}

	// Print connection status.
	fmt.Println("\tConnected:", common.FormatConnectionStatus(state.Connected))

	// Print content information.
	fmt.Printf("\tContents: %s | %s | %s\n",
		formatDirectoryCount(state.DirectoryCount),
		formatFileCountAndSize(state.FileCount, state.TotalFileSize),
		formatSymbolicLinkCount(state.SymbolicLinkCount),
	)

	// Print scan problems, if any.
	if len(state.ScanProblems) > 0 {
		if mode == common.SessionDisplayModeList {
			color.Red("\tScan problems: %d\n",
				uint64(len(state.ScanProblems))+state.ExcludedScanProblems,
			)
		} else if mode == common.SessionDisplayModeListLong {
			color.Red("\tScan problems:\n")
			for _, p := range state.ScanProblems {
				color.Red("\t\t%s: %v\n", formatPath(p.Path), p.Error)
			}
			if state.ExcludedScanProblems > 0 {
				color.Red("\t\t...+%d more...\n", state.ExcludedScanProblems)
			}
		}
	}

	// Print transition problems, if any.
	if len(state.TransitionProblems) > 0 {
		if mode == common.SessionDisplayModeList {
			color.Red("\tTransition problems: %d\n",
				uint64(len(state.TransitionProblems))+state.ExcludedTransitionProblems,
			)
		} else if mode == common.SessionDisplayModeListLong {
			color.Red("\tTransition problems:\n")
			for _, p := range state.TransitionProblems {
				color.Red("\t\t%s: %v\n", formatPath(p.Path), p.Error)
			}
			if state.ExcludedTransitionProblems > 0 {
				color.Red("\t\t...+%d more...\n", state.ExcludedTransitionProblems)
			}
		}
	}
}

// printConflictCount prints a count of synchronization conflicts.
func printConflictCount(conflicts []*core.Conflict, excludedConflicts uint64) {
	color.Red("Conflicts: %d\n", uint64(len(conflicts))+excludedConflicts)
}

// printConflicts prints a list of synchronization conflicts.
func printConflicts(conflicts []*core.Conflict, excludedConflicts uint64) {
	// Print the header.
	color.Red("Conflicts:\n")

	// Print conflicts.
	for i, c := range conflicts {
		// Print the alpha changes.
		for _, a := range c.AlphaChanges {
			color.Red(
				"\t(alpha) %s (%s -> %s)\n",
				formatPath(a.Path),
				formatEntry(a.Old),
				formatEntry(a.New),
			)
		}

		// Print the beta changes.
		for _, b := range c.BetaChanges {
			color.Red(
				"\t(beta)  %s (%s -> %s)\n",
				formatPath(b.Path),
				formatEntry(b.Old),
				formatEntry(b.New),
			)
		}

		// If we're not on the last conflict, or if there are conflicts that
		// have been excluded, then print a newline.
		if i < len(conflicts)-1 || excludedConflicts > 0 {
			fmt.Println()
		}
	}

	// Print excluded conflicts.
	if excludedConflicts > 0 {
		color.Red("\t...+%d more...\n", excludedConflicts)
	}
}

// printSession prints the configuration and status of a synchronization
// session and its endpoints.
func printSession(state *synchronization.State, mode common.SessionDisplayMode) {
	// Print name, if any.
	if state.Session.Name != "" {
		fmt.Println("Name:", state.Session.Name)
	}

	// Print the session identifier.
	fmt.Println("Identifier:", state.Session.Identifier)

	// Print extended information, if desired.
	if mode == common.SessionDisplayModeListLong || mode == common.SessionDisplayModeMonitorLong {
		// Print labels, if any.
		if len(state.Session.Labels) > 0 {
			fmt.Println("Labels:")
			keys := selection.ExtractAndSortLabelKeys(state.Session.Labels)
			for _, key := range keys {
				value := state.Session.Labels[key]
				if value == "" {
					value = emptyLabelValueDescription
				}
				fmt.Printf("\t%s: %s\n", key, value)
			}
		}

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
		symbolicLinkModeDescription := configuration.SymbolicLinkMode.Description()
		if configuration.SymbolicLinkMode.IsDefault() {
			defaultSymbolicLinkMode := state.Session.Version.DefaultSymbolicLinkMode()
			symbolicLinkModeDescription += fmt.Sprintf(" (%s)", defaultSymbolicLinkMode.Description())
		}
		fmt.Println("\tSymbolic link mode:", symbolicLinkModeDescription)

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
	}

	// Compute and print alpha-specific configuration.
	alphaConfigurationMerged := synchronization.MergeConfigurations(
		state.Session.Configuration,
		state.Session.ConfigurationAlpha,
	)
	printEndpoint(
		"Alpha", state.Session.Alpha,
		alphaConfigurationMerged, state.AlphaState,
		state.Session.Version,
		mode,
	)

	// Compute and print beta-specific configuration.
	betaConfigurationMerged := synchronization.MergeConfigurations(
		state.Session.Configuration,
		state.Session.ConfigurationBeta,
	)
	printEndpoint(
		"Beta", state.Session.Beta,
		betaConfigurationMerged, state.BetaState,
		state.Session.Version,
		mode,
	)

	// At this point, there's no other status information that will be displayed
	// for non-list modes, so we can save ourselves some checks and return if
	// we're in a monitor mode.
	if mode == common.SessionDisplayModeMonitor || mode == common.SessionDisplayModeMonitorLong {
		return
	}

	// Print conflicts, if any.
	if len(state.Conflicts) > 0 {
		if mode == common.SessionDisplayModeList {
			printConflictCount(state.Conflicts, state.ExcludedConflicts)
		} else if mode == common.SessionDisplayModeListLong {
			printConflicts(state.Conflicts, state.ExcludedConflicts)
		}
	}

	// Print the last error, if any.
	if state.LastError != "" {
		color.Red("Last error: %s\n", state.LastError)
	}

	// Print the session status .
	statusString := state.Status.Description()
	if state.Session.Paused {
		statusString = color.YellowString("[Paused]")
	}
	fmt.Fprintln(color.Output, "Status:", statusString)

	// Print staging progress if we're staging files and progress information is
	// available for the target endpoint.
	var stagingProgress *rsync.ReceiverState
	var totalExpectedSize uint64
	if state.Status == synchronization.Status_StagingAlpha {
		stagingProgress = state.AlphaState.StagingProgress
		if stagingProgress != nil && stagingProgress.ExpectedFiles == state.BetaState.FileCount {
			totalExpectedSize = state.BetaState.TotalFileSize
		}
	} else if state.Status == synchronization.Status_StagingBeta {
		stagingProgress = state.BetaState.StagingProgress
		if stagingProgress != nil && stagingProgress.ExpectedFiles == state.AlphaState.FileCount {
			totalExpectedSize = state.AlphaState.TotalFileSize
		}
	}
	if stagingProgress != nil {
		var fractionComplete float32
		var totalSizeDenominator string
		if totalExpectedSize != 0 {
			fractionComplete = float32(stagingProgress.TotalReceivedSize) / float32(totalExpectedSize)
			totalSizeDenominator = "/" + humanize.Bytes(totalExpectedSize)
		} else {
			fractionComplete = float32(stagingProgress.ReceivedFiles) / float32(stagingProgress.ExpectedFiles)
		}
		fmt.Printf("Staging progress: %d/%d - %s%s - %.0f%%\nCurrent file: %s (%s/%s)\n",
			stagingProgress.ReceivedFiles, stagingProgress.ExpectedFiles,
			humanize.Bytes(stagingProgress.TotalReceivedSize), totalSizeDenominator,
			100.0*fractionComplete,
			stagingProgress.Path,
			humanize.Bytes(stagingProgress.ReceivedSize), humanize.Bytes(stagingProgress.ExpectedSize),
		)
	}
}
