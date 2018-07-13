// +build !windows,!plan9

// TODO: Figure out what to do for Plan 9. It doesn't have syscall.Stat_t.

package filesystem

import (
	"os"
	"syscall"

	"github.com/pkg/errors"
)

// DeviceID extracts the device ID from a stat result.
func DeviceID(info os.FileInfo) (uint64, error) {
	// Grab the system-specific stat type.
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, errors.New("unable to extract raw filesystem information")
	}

	// Success.
	return uint64(stat.Dev), nil
}
