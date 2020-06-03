package forward

import (
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/forwarding"
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
func printEndpoint(name string, url *url.URL, configuration *forwarding.Configuration, version forwarding.Version) {
	// Print the endpoint header.
	fmt.Println(name, "configuration:")

	// Print the URL.
	fmt.Println("\tURL:", url.Format("\n\t\t"))

	// Compute and print the socket overwrite mode.
	socketOverwriteModeDescription := configuration.SocketOverwriteMode.Description()
	if configuration.SocketOverwriteMode.IsDefault() {
		socketOverwriteModeDescription += fmt.Sprintf(" (%s)", version.DefaultSocketOverwriteMode().Description())
	}
	fmt.Println("\tSocket overwrite mode:", socketOverwriteModeDescription)

	// Compute and print the socket owner.
	socketOwnerDescription := "Default"
	if configuration.SocketOwner != "" {
		socketOwnerDescription = configuration.SocketOwner
	}
	fmt.Println("\tSocket owner:", socketOwnerDescription)

	// Compute and print the socket group.
	socketGroupDescription := "Default"
	if configuration.SocketGroup != "" {
		socketGroupDescription = configuration.SocketGroup
	}
	fmt.Println("\tSocket group:", socketGroupDescription)

	// Compute and print the socket permission mode.
	var socketPermissionModeDescription string
	if configuration.SocketPermissionMode == 0 {
		socketPermissionModeDescription = fmt.Sprintf("Default (%#o)", version.DefaultSocketPermissionMode())
	} else {
		socketPermissionModeDescription = fmt.Sprintf("%#o", configuration.SocketPermissionMode)
	}
	fmt.Println("\tSocket permission mode:", socketPermissionModeDescription)
}

// printSession prints the configuration and status of a forwarding session and
// its endpoints.
func printSession(state *forwarding.State, long bool) {
	// Print name, if any.
	if state.Session.Name != "" {
		fmt.Println("Name:", state.Session.Name)
	}

	// Print the session identifier.
	fmt.Println("Identifier:", state.Session.Identifier)

	// Print labels.
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
	} else {
		fmt.Println("Labels: None")
	}

	// Print extended information, if desired.
	if long {
		// Print the configuration header.
		fmt.Println("Configuration:")

		// Compute and print source-specific configuration.
		sourceConfigurationMerged := forwarding.MergeConfigurations(
			state.Session.Configuration,
			state.Session.ConfigurationSource,
		)
		printEndpoint("Source", state.Session.Source, sourceConfigurationMerged, state.Session.Version)

		// Compute and print beta-specific configuration.
		destinationConfigurationMerged := forwarding.MergeConfigurations(
			state.Session.Configuration,
			state.Session.ConfigurationDestination,
		)
		printEndpoint("Destination", state.Session.Destination, destinationConfigurationMerged, state.Session.Version)
	}
}
