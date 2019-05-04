package filesystem

// probeExecutabilityPreservationFastByFormat checks if the specified format
// matches well-known executability preservation behavior.
func probeExecutabilityPreservationFastByFormat(format volumeFormat) (bool, bool) {
	switch format {
	case volumeFormatAPFS:
		return true, true
	case volumeFormatHFS:
		return true, true
	default:
		return false, false
	}
}

// probeExecutabilityPreservationFastByPath attempts to perform a fast
// executability preservation test by path, without probe files. The
// successfulness of the test is indicated by the second return parameter.
func probeExecutabilityPreservationFastByPath(path string) (bool, bool) {
	// Query the filesystem format.
	format, err := queryVolumeFormatByPath(path)
	if err != nil {
		return false, false
	}

	// Probe based on format.
	return probeExecutabilityPreservationFastByFormat(format)
}

// probeExecutabilityPreservationFast attempts to perform a fast executability
// preservation test, without probe files. The successfulness of the test is
// indicated by the second return parameter.
func probeExecutabilityPreservationFast(directory *Directory) (bool, bool) {
	// Query the filesystem format.
	format, err := queryVolumeFormat(directory)
	if err != nil {
		return false, false
	}

	// Probe based on format.
	return probeExecutabilityPreservationFastByFormat(format)
}
