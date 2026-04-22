package forward

import (
	"fmt"

	"github.com/dustin/go-humanize"

	"github.com/fatih/color"

	"github.com/mutagen-io/mutagen/cmd/mutagen/common"

	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/platform/terminal"
	"github.com/mutagen-io/mutagen/pkg/selection"
	"github.com/mutagen-io/mutagen/pkg/url"
)

const (
	// emptyLabelValueDescription is a human-friendly description representing
	// an empty label value. It contains characters which are invalid for use in
	// label values, so it won't be confused for one.
	emptyLabelValueDescription = "<empty>"
)

// printEndpoint prints the configuration for a forwarding endpoint.
func printEndpoint(name string, url *url.URL, configuration *forwarding.Configuration, state *forwarding.EndpointState, version forwarding.Version, mode common.SessionDisplayMode) {
	// Print the endpoint header.
	fmt.Printf("%s:\n", name)

	// Print the URL.
	fmt.Println("\tURL:", terminal.NeutralizeControlCharacters(url.Format("\n\t\t")))

	// Print parameters, if any.
	if len(url.Parameters) > 0 {
		fmt.Println("\tParameters:")
		keys := selection.ExtractAndSortLabelKeys(url.Parameters)
		for _, key := range keys {
			fmt.Printf("\t\t%s: %s\n", key, terminal.NeutralizeControlCharacters(url.Parameters[key]))
		}
	}

	// Print configuration information if desired.
	if mode == common.SessionDisplayModeListLong || mode == common.SessionDisplayModeMonitorLong {
		// Print configuration header.
		fmt.Println("\tConfiguration:")

		// Compute and print the socket overwrite mode.
		socketOverwriteModeDescription := configuration.SocketOverwriteMode.Description()
		if configuration.SocketOverwriteMode.IsDefault() {
			socketOverwriteModeDescription += fmt.Sprintf(" (%s)", version.DefaultSocketOverwriteMode().Description())
		}
		fmt.Println("\t\tSocket overwrite mode:", socketOverwriteModeDescription)

		// Compute and print the socket owner.
		socketOwnerDescription := "Default"
		if configuration.SocketOwner != "" {
			socketOwnerDescription = configuration.SocketOwner
		}
		fmt.Println("\t\tSocket owner:", socketOwnerDescription)

		// Compute and print the socket group.
		socketGroupDescription := "Default"
		if configuration.SocketGroup != "" {
			socketGroupDescription = configuration.SocketGroup
		}
		fmt.Println("\t\tSocket group:", socketGroupDescription)

		// Compute and print the socket permission mode.
		var socketPermissionModeDescription string
		if configuration.SocketPermissionMode == 0 {
			socketPermissionModeDescription = fmt.Sprintf("Default (%#o)", version.DefaultSocketPermissionMode())
		} else {
			socketPermissionModeDescription = fmt.Sprintf("%#o", configuration.SocketPermissionMode)
		}
		fmt.Println("\t\tSocket permission mode:", socketPermissionModeDescription)
	}

	// At this point, there's no other status information that will be displayed
	// for non-list modes, so we can save ourselves some checks and return if
	// we're in a monitor mode.
	if mode == common.SessionDisplayModeMonitor || mode == common.SessionDisplayModeMonitorLong {
		return
	}

	// Print connection status.
	fmt.Println("\tConnected:", common.FormatConnectionStatus(state.Connected))
}

// printSession prints the configuration and status of a forwarding session and
// its endpoints.
func printSession(state *forwarding.State, mode common.SessionDisplayMode) {
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

		// Print the configuration header and configuration.
		// TODO: Implement this in the future if we have any session-level
		// configuration behaviors. We currently only have endpoint-level
		// configuration behaviors and thus we don't print any session-level
		// configuration information like we do for synchronization.
	}

	// Compute and print source-specific configuration.
	sourceConfigurationMerged := forwarding.MergeConfigurations(
		state.Session.Configuration,
		state.Session.ConfigurationSource,
	)
	printEndpoint(
		"Source", state.Session.Source,
		sourceConfigurationMerged, state.SourceState,
		state.Session.Version,
		mode,
	)

	// Compute and print beta-specific configuration.
	destinationConfigurationMerged := forwarding.MergeConfigurations(
		state.Session.Configuration,
		state.Session.ConfigurationDestination,
	)
	printEndpoint(
		"Destination", state.Session.Destination,
		destinationConfigurationMerged, state.DestinationState,
		state.Session.Version,
		mode,
	)

	// At this point, there's no other status information that will be displayed
	// for non-list modes, so we can save ourselves some checks and return if
	// we're in a monitor mode.
	if mode == common.SessionDisplayModeMonitor || mode == common.SessionDisplayModeMonitorLong {
		return
	}

	// Print the last error, if any.
	if state.LastError != "" {
		color.Red("Last error: %s\n", terminal.NeutralizeControlCharacters(state.LastError))
	}

	// Print the session status .
	statusString := state.Status.Description()
	if state.Session.Paused {
		statusString = color.YellowString("[Paused]")
	}
	fmt.Fprintln(color.Output, "Status:", statusString)

	// Print connection statistics if we're forwarding.
	if state.Status == forwarding.Status_ForwardingConnections {
		fmt.Printf("Connections: %d open, %d total, %s outbound, %s inbound\n",
			state.OpenConnections,
			state.TotalConnections,
			humanize.Bytes(state.TotalOutboundData),
			humanize.Bytes(state.TotalInboundData),
		)
	}
}
