package sync

func iterate(contentLists ...map[string]*Entry) map[string]bool {
	// Compute the maximum length.
	maxLength := 0
	for _, l := range contentLists {
		if maxLength < len(l) {
			maxLength = len(l)
		}
	}

	// If there are no entries in any list, save an allocation.
	if maxLength == 0 {
		return nil
	}

	// Create and populate the union map. We create the map with the minimum
	// necessary capacity, but it should be a sufficient value for most cases
	// when nodes are equal.
	result := make(map[string]bool, maxLength)
	for _, c := range contentLists {
		for n, _ := range c {
			result[n] = true
		}
	}

	// Done.
	return result
}
