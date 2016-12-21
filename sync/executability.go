package sync

func propagateExecutability(ancestor, snapshot *Entry) {
	// If either entry is nil or their types don't match, then there's nothing
	// to propagate.
	if ancestor == nil || snapshot == nil || ancestor.Kind != snapshot.Kind {
		return
	}

	// Handle the propagation based on entry kind.
	if snapshot.Kind == EntryKind_File {
		snapshot.Executable = ancestor.Executable
	} else if snapshot.Kind == EntryKind_Directory {
		iterate2(ancestor.GetContents(), snapshot.GetContents(), func(_ string, a, s *Entry) {
			propagateExecutability(a, s)
		})
	} else {
		panic("unhandled entry kind")
	}
}

func PropagateExecutability(ancestor, snapshot *Entry) *Entry {
	// Create a copy of the snapshot that we can mutate.
	result := snapshot.copy()

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
	if snapshot.Kind == EntryKind_File {
		snapshot.Executable = false
	} else if snapshot.Kind == EntryKind_Directory {
		for _, sc := range snapshot.Contents {
			stripExecutability(sc.Entry)
		}
	} else {
		panic("unhandled entry kind")
	}
}

func StripExecutability(snapshot *Entry) *Entry {
	// Create a copy of the snapshot that we can mutate.
	result := snapshot.copy()

	// Perform stripping.
	stripExecutability(result)

	// Done.
	return result
}
