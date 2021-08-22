//go:build darwin || linux

package format

import (
	"fmt"

	"golang.org/x/sys/unix"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

// statfsRetryingOnEINTR is a wrapper around the statfs system call that retries
// on EINTR errors and returns on the first successful call or non-EINTR error.
func statfsRetryingOnEINTR(path string, metadata *unix.Statfs_t) error {
	for {
		err := unix.Statfs(path, metadata)
		if err == unix.EINTR {
			continue
		}
		return err
	}
}

// fstatfsRetryingOnEINTR is a wrapper around the fstatfs system call that
// retries on EINTR errors and returns on the first successful call or non-EINTR
// error.
func fstatfsRetryingOnEINTR(fd int, metadata *unix.Statfs_t) error {
	for {
		err := unix.Fstatfs(fd, metadata)
		if err == unix.EINTR {
			continue
		}
		return err
	}
}

// QueryByPath queries the filesystem format for the specified path.
func QueryByPath(path string) (Format, error) {
	// Perform a filesystem metadata query on the path.
	var metadata unix.Statfs_t
	if err := statfsRetryingOnEINTR(path, &metadata); err != nil {
		return FormatUnknown, fmt.Errorf("unable to query filesystem metadata: %w", err)
	}

	// Classify the filesystem.
	return formatFromStatfs(&metadata), nil
}

// Query queries the filesystem format for the specified directory.
func Query(directory *filesystem.Directory) (Format, error) {
	// Perform a filesystem metadata query on the directory.
	var metadata unix.Statfs_t
	if err := fstatfsRetryingOnEINTR(directory.Descriptor(), &metadata); err != nil {
		return FormatUnknown, fmt.Errorf("unable to query filesystem metadata: %w", err)
	}

	// Classify the filesystem.
	return formatFromStatfs(&metadata), nil
}
