// +build darwin linux

package filesystem

// probeUnicodeDecompositionFast attempts to perform a fast Unicode
// decomposition test by path, without probe files. The successfulness of the
// test is indicated by the second return parameter.
func probeUnicodeDecompositionFastByPath(path string) (bool, bool) {
	if f, err := QueryFormatByPath(path); err != nil {
		return false, false
	} else {
		return probeUnicodeDecompositionFastByFormat(f)
	}
}

// probeUnicodeDecompositionFast attempts to perform a fast Unicode
// decomposition test, without probe files. The successfulness of the test is
// indicated by the second return parameter.
func probeUnicodeDecompositionFast(directory *Directory) (bool, bool) {
	if f, err := QueryFormat(directory); err != nil {
		return false, false
	} else {
		return probeUnicodeDecompositionFastByFormat(f)
	}
}
