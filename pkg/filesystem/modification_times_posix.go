// +build !windows,!darwin,!netbsd

package filesystem

import (
	"golang.org/x/sys/unix"
)

// extractModificationTime is a convenience function for extracting the
// modification time specification from a Stat_t structure. It's necessary since
// not all POSIX platforms use the same struct field name for this value.
func extractModificationTime(metadata *unix.Stat_t) unix.Timespec {
	return metadata.Mtim
}
