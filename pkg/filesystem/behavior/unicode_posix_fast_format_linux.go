package behavior

import (
	"github.com/havoc-io/mutagen/pkg/filesystem"
)

// probeUnicodeDecompositionFastByFormat checks if the specified format matches
// well-known Unicode decomposition behavior.
func probeUnicodeDecompositionFastByFormat(format filesystem.Format) (bool, bool) {
	switch format {
	case filesystem.FormatEXT:
		return false, true
	case filesystem.FormatNFS:
		return false, true
	default:
		return false, false
	}
}
