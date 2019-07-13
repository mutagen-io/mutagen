package forwarding

import (
	"fmt"

	forwardingpkg "github.com/havoc-io/mutagen/pkg/forwarding"
	"github.com/havoc-io/mutagen/pkg/selection"
	urlpkg "github.com/havoc-io/mutagen/pkg/url"
)

const (
	// emptyLabelValueDescription is a human-friendly description representing
	// an empty label value. It contains characters which are invalid for use in
	// label values, so it won't be confused for one.
	emptyLabelValueDescription = "<empty>"
)

func printEndpoint(name string, url *urlpkg.URL, configuration *forwardingpkg.Configuration, version forwardingpkg.Version) {
	// Print the endpoint header.
	fmt.Println(name, "configuration:")

	// Print the URL.
	fmt.Println("\tURL:", url.Format("\n\t\t"))
}

func printSession(state *forwardingpkg.State) {
	// Print the session identifier.
	fmt.Println("Session:", state.Session.Identifier)

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
}
