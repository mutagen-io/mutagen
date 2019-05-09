package behavior

import (
	"github.com/havoc-io/mutagen/pkg/filesystem"
)

// probeExecutabilityPreservationFastByPath attempts to perform a fast
// executability preservation test by path, without probe files. The
// successfulness of the test is indicated by the second return parameter.
func probeExecutabilityPreservationFastByPath(_ string) (bool, bool) {
	return false, true
}

// probeExecutabilityPreservationFast attempts to perform a fast executability
// preservation test, without probe files. The successfulness of the test is
// indicated by the second return parameter.
func probeExecutabilityPreservationFast(_ *filesystem.Directory) (bool, bool) {
	return false, true
}
