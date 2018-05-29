package sync

func propagateExecutability(ancestor, snapshot *Entry) {
	// If either entry is nil or their types don't match, then there's nothing
	// to propagate.
	if ancestor == nil || snapshot == nil || ancestor.Kind != snapshot.Kind {
		return
	}

	// Handle the propagation based on entry kind.
	if snapshot.Kind == EntryKind_Directory {
		ancestorContents := ancestor.GetContents()
		snapshotContents := snapshot.GetContents()
		for name := range nameUnion(ancestorContents, snapshotContents) {
			propagateExecutability(ancestorContents[name], snapshotContents[name])
		}
	} else if snapshot.Kind == EntryKind_File {
		snapshot.Executable = ancestor.Executable
	}
}

func PropagateExecutability(ancestor, snapshot *Entry) *Entry {
	// Create a copy of the snapshot that we can mutate.
	result := snapshot.Copy()

	// Perform propagation.
	propagateExecutability(ancestor, result)

	// Done.
	return result
}

func stripExecutability(snapshot *Entry) {
	// If the entry is nil, then there's nothing to strip.
	if snapshot == nil {
		return
	}

	// Handle the propagation based on entry kind.
	if snapshot.Kind == EntryKind_Directory {
		for _, entry := range snapshot.Contents {
			stripExecutability(entry)
		}
	} else if snapshot.Kind == EntryKind_File {
		snapshot.Executable = false
	}
}

func StripExecutability(snapshot *Entry) *Entry {
	// Create a copy of the snapshot that we can mutate.
	result := snapshot.Copy()

	// Perform stripping.
	stripExecutability(result)

	// Done.
	return result
}
