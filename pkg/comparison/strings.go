package comparison

// StringSlicesEqual determines whether or not two string slices are equal. It
// does not consider nilness when comparing zero-length slices.
func StringSlicesEqual(first, second []string) bool {
	// Check that slice lengths are equal.
	if len(first) != len(second) {
		return false
	}

	// Compare contents.
	for i, f := range first {
		if second[i] != f {
			return false
		}
	}

	// The slices are equal.
	return true
}

// StringMapsEqual determines whether or not two string maps are equal. It does
// not consider nilness when comparing zero-length maps.
func StringMapsEqual(first, second map[string]string) bool {
	// Check that map lengths are equal.
	if len(first) != len(second) {
		return false
	}

	// Compare contents.
	for key, f := range first {
		if s, ok := second[key]; !ok || s != f {
			return false
		}
	}

	// The maps are equal.
	return true
}
