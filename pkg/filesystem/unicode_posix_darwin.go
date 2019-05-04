package filesystem

// probeUnicodeDecompositionFastByFormat checks if the specified format matches
// well-known Unicode decomposition behavior.
func probeUnicodeDecompositionFastByFormat(format volumeFormat) (bool, bool) {
	switch format {
	case volumeFormatAPFS:
		return false, true
	case volumeFormatHFS:
		return true, true
	default:
		return false, false
	}
}

// probeUnicodeDecompositionFast attempts to perform a fast Unicode
// decomposition test by path, without probe files. The successfulness of the
// test is indicated by the second return parameter.
func probeUnicodeDecompositionFastByPath(path string) (bool, bool) {
	// Query the filesystem format.
	format, err := queryVolumeFormatByPath(path)
	if err != nil {
		return false, false
	}

	// Probe based on format.
	return probeUnicodeDecompositionFastByFormat(format)
}

// probeUnicodeDecompositionFast attempts to perform a fast Unicode
// decomposition test, without probe files. The successfulness of the test is
// indicated by the second return parameter.
func probeUnicodeDecompositionFast(directory *Directory) (bool, bool) {
	// Query the filesystem format.
	format, err := queryVolumeFormat(directory)
	if err != nil {
		return false, false
	}

	// Probe based on format.
	return probeUnicodeDecompositionFastByFormat(format)
}
