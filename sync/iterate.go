package sync

func iterate2(first, second []*NamedEntry, action func(string, *Entry, *Entry)) {
	iterate3(first, second, nil, func(name string, f, s, _ *Entry) {
		action(name, f, s)
	})
}

func iterate3(first, second, third []*NamedEntry, action func(string, *Entry, *Entry, *Entry)) {
	// Iterate while there are contents remaining.
	for len(first) > 0 || len(second) > 0 || len(third) > 0 {
		// Compute the target name.
		name := ""
		if len(first) > 0 && (name == "" || first[0].Name < name) {
			name = first[0].Name
		}
		if len(second) > 0 && (name == "" || second[0].Name < name) {
			name = second[0].Name
		}
		if len(third) > 0 && (name == "" || third[0].Name < name) {
			name = third[0].Name
		}

		// Extract entries and reduce lists.
		var f, s, t *Entry
		if len(first) > 0 && first[0].Name == name {
			f = first[0].Entry
			first = first[1:]
		}
		if len(second) > 0 && second[0].Name == name {
			s = second[0].Entry
			second = second[1:]
		}
		if len(third) > 0 && third[0].Name == name {
			t = third[0].Entry
			third = third[1:]
		}

		// Invoke the callback.
		action(name, f, s, t)
	}
}
