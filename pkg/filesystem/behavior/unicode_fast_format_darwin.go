package behavior

import (
	"github.com/havoc-io/mutagen/pkg/filesystem/behavior/internal/format"
)

// probeUnicodeDecompositionFastByFormat checks if the specified format matches
// well-known Unicode decomposition behavior.
func probeUnicodeDecompositionFastByFormat(f format.Format) (bool, bool) {
	switch f {
	case format.FormatAPFS:
		return false, true
	case format.FormatHFS:
		return true, true
	default:
		return false, false
	}
}
