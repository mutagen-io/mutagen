// +build darwin linux

package filesystem

// probeExecutabilityPreservationFastByPath attempts to perform a fast
// executability preservation test by path, without probe files. The
// successfulness of the test is indicated by the second return parameter.
func probeExecutabilityPreservationFastByPath(path string) (bool, bool) {
	if f, err := QueryFormatByPath(path); err != nil {
		return false, false
	} else {
		return probeExecutabilityPreservationFastByFormat(f)
	}
}

// probeExecutabilityPreservationFast attempts to perform a fast executability
// preservation test, without probe files. The successfulness of the test is
// indicated by the second return parameter.
func probeExecutabilityPreservationFast(directory *Directory) (bool, bool) {
	if f, err := QueryFormat(directory); err != nil {
		return false, false
	} else {
		return probeExecutabilityPreservationFastByFormat(f)
	}
}
