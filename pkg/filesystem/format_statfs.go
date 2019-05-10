// +build darwin linux

package filesystem

import (
	"github.com/pkg/errors"

	"golang.org/x/sys/unix"
)

// QueryFormatByPath queries the filesystem format for the specified path.
func QueryFormatByPath(path string) (Format, error) {
	// Perform a filesystem metadata query on the path.
	var metadata unix.Statfs_t
	if err := unix.Statfs(path, &metadata); err != nil {
		return FormatUnknown, errors.Wrap(err, "unable to query filesystem metadata")
	}

	// Classify the filesystem.
	return formatFromStatfs(&metadata), nil
}

// QueryFormat queries the filesystem format for the specified directory.
func QueryFormat(directory *Directory) (Format, error) {
	// Perform a filesystem metadata query on the directory.
	var metadata unix.Statfs_t
	if err := unix.Fstatfs(directory.descriptor, &metadata); err != nil {
		return FormatUnknown, errors.Wrap(err, "unable to query filesystem metadata")
	}

	// Classify the filesystem.
	return formatFromStatfs(&metadata), nil
}
