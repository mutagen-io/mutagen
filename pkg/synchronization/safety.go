package synchronization

import (
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
)

// oneEndpointEmptiedRoot determines whether or not one endpoint (but not both)
// transitioned from a directory root with two or more content entries to a
// directory root without any content.
func oneEndpointEmptiedRoot(ancestor, αSnapshot, βSnapshot *core.Entry) bool {
	// Check if alpha emptied root content.
	αEmptiedRoot := ancestor != nil && αSnapshot != nil &&
		ancestor.Kind == core.EntryKind_Directory &&
		αSnapshot.Kind == core.EntryKind_Directory &&
		len(ancestor.Contents) >= 2 && len(αSnapshot.Contents) == 0

	// Check if beta emptied root content.
	βEmptiedRoot := ancestor != nil && βSnapshot != nil &&
		ancestor.Kind == core.EntryKind_Directory &&
		βSnapshot.Kind == core.EntryKind_Directory &&
		len(ancestor.Contents) >= 2 && len(βSnapshot.Contents) == 0

	// Determine whether one (and only one) endpoint emptied root content.
	return (αEmptiedRoot || βEmptiedRoot) && !(αEmptiedRoot && βEmptiedRoot)
}

// isRootDeletion determines whether or not the specified change is a root
// deletion.
func isRootDeletion(change *core.Change) bool {
	return change.Path == "" && change.Old != nil && change.New == nil
}

// isRootTypeChange determines whether or not the specified change is a root
// type change.
func isRootTypeChange(change *core.Change) bool {
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
