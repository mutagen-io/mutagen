package behavior

import (
	"github.com/havoc-io/mutagen/pkg/filesystem"
)

// probeUnicodeDecompositionFast attempts to perform a fast Unicode
// decomposition test by path, without probe files. The successfulness of the
// test is indicated by the second return parameter.
func probeUnicodeDecompositionFastByPath(_ string) (bool, bool) {
	return false, true
}

// probeUnicodeDecompositionFast attempts to perform a fast Unicode
// decomposition test, without probe files. The successfulness of the test is
// indicated by the second return parameter.
func probeUnicodeDecompositionFast(_ *filesystem.Directory) (bool, bool) {
	return false, true
}
