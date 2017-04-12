package sync

func nameUnion(contentMaps ...map[string]*Entry) map[string]bool {
	// Create the result. As a very rough but fast heuristic, we use the size of
	// the first map as an estimate of the required capacity. For most cases,
	// where all maps have the same contents due to a lack of changes, this
	// should provide savings due to map reallocations.
	capacity := 0
	if len(contentMaps) > 0 {
		capacity = len(contentMaps[0])
	}
	result := make(map[string]bool, capacity)

	// Populate it.
	for _, contents := range contentMaps {
		for name, _ := range contents {
			result[name] = true
		}
	}

	// Done.
	return result
}
