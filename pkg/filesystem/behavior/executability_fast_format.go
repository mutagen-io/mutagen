// +build darwin linux

package behavior

import (
	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/filesystem/behavior/internal/format"
)

// probeExecutabilityPreservationFastByPath attempts to perform a fast
// executability preservation test by path, without probe files. The
// successfulness of the test is indicated by the second return parameter.
func probeExecutabilityPreservationFastByPath(path string) (bool, bool) {
	if f, err := format.QueryByPath(path); err != nil {
		return false, false
	} else {
		return probeExecutabilityPreservationFastByFormat(f)
	}
}

// probeExecutabilityPreservationFast attempts to perform a fast executability
// preservation test, without probe files. The successfulness of the test is
// indicated by the second return parameter.
func probeExecutabilityPreservationFast(directory *filesystem.Directory) (bool, bool) {
	if f, err := format.Query(directory); err != nil {
		return false, false
	} else {
		return probeExecutabilityPreservationFastByFormat(f)
	}
}
