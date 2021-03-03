package utility

// CopyStringSlice creates a copy of a string slice. It preserves nil/non-nil
// characteristics for empty slices.
func CopyStringSlice(s []string) []string {
	// If the slice is nil, then preserve its nilness. For zero-length, non-nil
	// slices, we still allocate on the heap to preserve non-nilness.
	if s == nil {
		return nil
	}

	// Make a copy.
	result := make([]string, len(s))
	copy(result, s)

	// Done.
	return result
}

// CopyStringMap creates a copy of a string map. It preserves nil/non-nil
// characteristics for empty maps.
func CopyStringMap(m map[string]string) map[string]string {
	// If the map is nil, then preserve its nilness. For zero-length, non-nil
	// maps, we still allocate on the heap to preserve non-nilness.
	if m == nil {
		return nil
	}

	// Make a copy.
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}

	// Done.
	return result
}
