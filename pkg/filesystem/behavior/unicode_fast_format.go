// +build darwin linux

package behavior

import (
	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/filesystem/behavior/internal/format"
)

// probeUnicodeDecompositionFast attempts to perform a fast Unicode
// decomposition test by path, without probe files. The successfulness of the
// test is indicated by the second return parameter.
func probeUnicodeDecompositionFastByPath(path string) (bool, bool) {
	if f, err := format.QueryByPath(path); err != nil {
		return false, false
	} else {
		return probeUnicodeDecompositionFastByFormat(f)
	}
}

// probeUnicodeDecompositionFast attempts to perform a fast Unicode
// decomposition test, without probe files. The successfulness of the test is
// indicated by the second return parameter.
func probeUnicodeDecompositionFast(directory *filesystem.Directory) (bool, bool) {
	if f, err := format.Query(directory); err != nil {
		return false, false
	} else {
		return probeUnicodeDecompositionFastByFormat(f)
	}
}
