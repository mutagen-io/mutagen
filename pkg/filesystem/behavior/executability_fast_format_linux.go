package behavior

import (
	"github.com/havoc-io/mutagen/pkg/filesystem"
)

// probeExecutabilityPreservationFastByFormat checks if the specified format
// matches well-known executability preservation behavior.
func probeExecutabilityPreservationFastByFormat(format filesystem.Format) (bool, bool) {
	switch format {
	case filesystem.FormatEXT:
		return true, true
	case filesystem.FormatNFS:
		return true, true
	default:
		return false, false
	}
}
