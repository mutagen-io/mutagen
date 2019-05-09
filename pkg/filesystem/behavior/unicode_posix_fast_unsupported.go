// +build !windows,!darwin,!linux

package behavior

import (
	"github.com/havoc-io/mutagen/pkg/filesystem"
)

// probeUnicodeDecompositionFast attempts to perform a fast Unicode
// decomposition test by path, without probe files. The successfulness of the
// test is indicated by the second return parameter.
func probeUnicodeDecompositionFastByPath(path string) (bool, bool) {
	return false, false
}

// probeUnicodeDecompositionFast attempts to perform a fast Unicode
// decomposition test, without probe files. The successfulness of the test is
// indicated by the second return parameter.
func probeUnicodeDecompositionFast(directory *filesystem.Directory) (bool, bool) {
	return false, false
}
