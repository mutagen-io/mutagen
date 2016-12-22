package sync

func nameUnion(contentMaps ...map[string]*Entry) map[string]bool {
	// Create the result.
	result := make(map[string]bool)

	// Populate it.
	for _, contents := range contentMaps {
		for name, _ := range contents {
			result[name] = true
		}
	}

	// Done.
	return result
}
