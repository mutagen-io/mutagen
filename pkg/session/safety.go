package session

import (
	"github.com/havoc-io/mutagen/pkg/sync"
)

// isRootDeletion determines whether or not the specified change is a root
// deletion.
func isRootDeletion(change *sync.Change) bool {
	return change.Path == "" && change.Old != nil && change.New == nil
}

// isRootTypeChange determines whether or not the specified change is a root
// type change.
func isRootTypeChange(change *sync.Change) bool {
	return change.Path == "" &&
		change.Old != nil && change.New != nil &&
		change.Old.Kind != change.New.Kind
}

// filteredPathsAreSubset checks whether or not a slice of filtered paths is a
// subset of a larger slice of unfiltered paths. The paths in the filtered slice
// must share the same relative ordering as in the original slice.
func filteredPathsAreSubset(filteredPaths, originalPaths []string) bool {
	// Loop over the list of filtered paths.
	for _, filtered := range filteredPaths {
		// Track whether or not we find a match for this path in what remains of
		// the original path list.
		matched := false

		// Loop over what remains of the original paths.
		for o, original := range originalPaths {
			if original == filtered {
				originalPaths = originalPaths[o+1:]
				matched = true
				break
			}
		}

		// Check if we found a match.
		if !matched {
			return false
		}
	}

	// Success.
	return true
}
