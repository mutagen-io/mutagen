package filesystem

// probeUnicodeDecompositionFastByFormat checks if the specified format matches
// well-known Unicode decomposition behavior.
func probeUnicodeDecompositionFastByFormat(f Format) (bool, bool) {
	switch f {
	case FormatAPFS:
		return false, true
	case FormatHFS:
		return true, true
	default:
		return false, false
	}
}
