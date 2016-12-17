package sync

func PropagateExecutability(ancestor, snapshot *Entry) {
	// If either entry is nil or their kinds don't match, there's nothing to
	// propagate.
	if ancestor == nil || snapshot == nil || ancestor.Kind != snapshot.Kind {
		return
	}

	// Handle the propagation based on entry kind.
	if snapshot.Kind == EntryKind_File {
		snapshot.Executable = ancestor.Executable
	} else if snapshot.Kind == EntryKind_Directory {
		for _, sc := range snapshot.Contents {
			if a, ok := ancestor.Find(sc.Name); ok {
				PropagateExecutability(a, sc.Entry)
			}
		}
	} else {
		panic("unhandled entry kind")
	}
}
