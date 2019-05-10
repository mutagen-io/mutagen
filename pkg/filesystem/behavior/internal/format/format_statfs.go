// +build darwin linux

package format

import (
	"github.com/pkg/errors"

	"golang.org/x/sys/unix"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

// QueryByPath queries the filesystem format for the specified path.
func QueryByPath(path string) (Format, error) {
	// Perform a filesystem metadata query on the path.
	var metadata unix.Statfs_t
	if err := unix.Statfs(path, &metadata); err != nil {
		return FormatUnknown, errors.Wrap(err, "unable to query filesystem metadata")
	}

	// Classify the filesystem.
	return formatFromStatfs(&metadata), nil
}

// Query queries the filesystem format for the specified directory.
func Query(directory *filesystem.Directory) (Format, error) {
	// Perform a filesystem metadata query on the directory.
	var metadata unix.Statfs_t
	if err := unix.Fstatfs(directory.Descriptor(), &metadata); err != nil {
		return FormatUnknown, errors.Wrap(err, "unable to query filesystem metadata")
	}

	// Classify the filesystem.
	return formatFromStatfs(&metadata), nil
}
