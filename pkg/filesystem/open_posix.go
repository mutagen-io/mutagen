//go:build !windows

package filesystem

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"golang.org/x/sys/unix"
)

// Open opens a filesystem path for traversal and/or other operations. It will
// return either a Directory or an io.ReadSeekCloser object (as an io.Closer for
// convenient closing access without casting), along with Metadata that can be
// used to determine the type of object being returned. Unless requested, this
// function does not allow the leaf component of path to be a symbolic link
// (though intermediate components of the path can be symbolic links and will be
// resolved in the resolution of the path), and an error will be returned if
// this is the case (though on POSIX systems it will not be
// ErrUnsupportedOpenType). However, if allowSymbolicLinkLeaf is true, then this
// function will allow resolution of a path leaf component that's a symbolic
// link. In this case, the referenced object must still be a directory or
// regular file, and the returned object will still be either a Directory or an
// io.ReadSeekCloser.
func Open(path string, allowSymbolicLinkLeaf bool, logger *logging.Logger) (io.Closer, *Metadata, error) {
	// Open the file. Unless explicitly allowed, we disable resolution of
	// symbolic links at the leaf position of the path by specifying O_NOFOLLOW.
	// Note that this flag only affects the leaf component of the path -
	// intermediate symbolic links are still allowed and resolved.
	//
	// Ideally, we'd want the open function to open the symbolic link itself
	// in the event that O_NOFOLLOW is specified and the path leaf references a
	// symbolic link, and then we could get stat information on the opened
	// symbolic link object and return ErrUnsupportedType. Unfortunately, that's
	// not what happens when you pass O_NOFOLLOW to open and the path leaf
	// references a symbolic link. Instead, it returns ELOOP. This is
	// problematic because ELOOP is also used to indicate the condition where
	// too many symbolic links have been encountered, and thus there's no way to
	// differentiate the two cases and figure out whether or not we should
	// return ErrUnsupportedType. Even openat doesn't provide a solution to this
	// problem since it doesn't support AT_SYMLINK_NOFOLLOW. Essentially,
	// there's no way to "open" a symbolic link - it can only be read with
	// readlink and its ilk. Since ELOOP still sort of makes sense (we've
	// encountered too many symbolic links at the path leaf), we return it
	// unmodified.
	flags := unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_CLOEXEC | extraOpenFlags
	if allowSymbolicLinkLeaf {
		flags &^= unix.O_NOFOLLOW
	}
	descriptor, err := openatRetryingOnEINTR(unix.AT_FDCWD, path, flags, 0)
	if err != nil {
		return nil, nil, err
	}

	// Grab metadata for the file.
	var rawMetadata unix.Stat_t
	if err := fstatRetryingOnEINTR(descriptor, &rawMetadata); err != nil {
		mustCloseConsideringEINTR(descriptor, logger)
		return nil, nil, fmt.Errorf("unable to query file metadata: %w", err)
	}

	// Convert the raw system-level metadata.
	metadata := &Metadata{
		Name:             filepath.Base(path),
		Mode:             Mode(rawMetadata.Mode),
		Size:             uint64(rawMetadata.Size),
		ModificationTime: time.Unix(rawMetadata.Mtim.Unix()),
		DeviceID:         uint64(rawMetadata.Dev),
		FileID:           uint64(rawMetadata.Ino),
	}

	// Dispatch further construction according to type.
	switch metadata.Mode & ModeTypeMask {
	case ModeTypeDirectory:
		return &Directory{
			descriptor: descriptor,
			file:       os.NewFile(uintptr(descriptor), path),
		}, metadata, nil
	case ModeTypeFile:
		return file(descriptor), metadata, nil
	default:
		mustCloseConsideringEINTR(descriptor, logger)
		return nil, nil, ErrUnsupportedOpenType
	}
}
