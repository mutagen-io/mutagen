package sync

import (
	"bytes"
)

// propagateExecutabilityRecursive propagates executability recursively.
func propagateExecutabilityRecursive(ancestor, source, target *Entry) {
	// If target is nil, then we don't have anything to propagate to, so bail.
	if target == nil {
		return
	}

	// Handle based on target kind.
	if target.Kind == EntryKind_Directory {
		ancestorContents := ancestor.GetContents()
		sourceContents := source.GetContents()
		targetContents := target.GetContents()
		for name := range targetContents {
			propagateExecutabilityRecursive(ancestorContents[name], sourceContents[name], targetContents[name])
		}
	} else if target.Kind == EntryKind_File {
		if source != nil && source.Kind == EntryKind_File && bytes.Equal(source.Digest, target.Digest) {
			target.Executable = source.Executable
		} else if ancestor != nil && ancestor.Kind == EntryKind_File && bytes.Equal(ancestor.Digest, target.Digest) {
			target.Executable = ancestor.Executable
		}
	}
}

// PropagateExecutability propagates file executability from the ancestor and
// source to the target in a recursive fashion. Executability information is
// only propagated if entry paths, types, and contents match, with source taking
// precedent over ancestor.
func PropagateExecutability(ancestor, source, target *Entry) *Entry {
	// Create a copy of the snapshot that we can mutate.
	result := target.Copy()

	// Perform propagation.
	propagateExecutabilityRecursive(ancestor, source, result)

	// Done.
	return result
}
