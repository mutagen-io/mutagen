package tunnel

import (
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/selection"
	"github.com/mutagen-io/mutagen/pkg/tunneling"
)

const (
	// emptyLabelValueDescription is a human-friendly description representing
	// an empty label value. It contains characters which are invalid for use in
	// label values, so it won't be confused for one.
	emptyLabelValueDescription = "<empty>"
)

// printTunnel prints the configuration for a tunnel.
func printTunnel(state *tunneling.State, long bool) {
	// Print name, if any.
	if state.Tunnel.Name != "" {
		fmt.Println("Name:", state.Tunnel.Name)
	}

	// Print the session identifier.
	fmt.Println("Identifier:", state.Tunnel.Identifier)

	// Print labels.
	if len(state.Tunnel.Labels) > 0 {
		fmt.Println("Labels:")
		keys := selection.ExtractAndSortLabelKeys(state.Tunnel.Labels)
		for _, key := range keys {
			value := state.Tunnel.Labels[key]
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

		// TODO: Eventually print tunnel configuration, if there is any.
	}
}
