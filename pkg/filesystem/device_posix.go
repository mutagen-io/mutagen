// +build !windows

package filesystem

import (
	"os"
	"syscall"

	"github.com/pkg/errors"
)

func DeviceID(path string) (uint64, error) {
	// Perform a stat on the path.
	info, err := os.Lstat(path)
	if err != nil {
		return 0, errors.Wrap(err, "unable to query filesystem information")
	}

	// Grab the system-specific stat type.
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, errors.Wrap(err, "unable to extract raw filesystem information")
	}

	// Success.
	return uint64(stat.Dev), nil
}
