package behavior

import (
	"github.com/havoc-io/mutagen/pkg/filesystem/behavior/internal/format"
)

// probeUnicodeDecompositionFastByFormat checks if the specified format matches
// well-known Unicode decomposition behavior.
func probeUnicodeDecompositionFastByFormat(f format.Format) (bool, bool) {
	switch f {
	case format.FormatEXT:
		return false, true
	case format.FormatNFS:
		return false, true
	default:
		return false, false
	}
}
