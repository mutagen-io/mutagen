package filesystem

// probeUnicodeDecompositionFastByFormat checks if the specified format matches
// well-known Unicode decomposition behavior.
func probeUnicodeDecompositionFastByFormat(f Format) (bool, bool) {
	switch f {
	case FormatEXT:
		return false, true
	case FormatNFS:
		return false, true
	default:
		return false, false
	}
}
