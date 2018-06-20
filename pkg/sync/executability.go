package sync

func propagateExecutability(ancestor, source, target *Entry) {
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
			propagateExecutability(ancestorContents[name], sourceContents[name], targetContents[name])
		}
	} else if target.Kind == EntryKind_File {
		if source != nil && source.Kind == EntryKind_File {
			target.Executable = source.Executable
		} else if ancestor != nil && ancestor.Kind == EntryKind_File {
			target.Executable = ancestor.Executable
		}
	}
}

func PropagateExecutability(ancestor, source, target *Entry) *Entry {
	// Create a copy of the snapshot that we can mutate.
	result := target.Copy()

	// Perform propagation.
	propagateExecutability(ancestor, source, result)

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
