// +build !windows

package filesystem

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pkg/errors"

	"golang.org/x/sys/unix"
)

// Open opens a filesystem path for traversal and operations. It will return
// either a Directory or ReadableFile object (as an io.Closer for convenient
// closing access without casting), along with Metadata that can be used to
// determine the type of object being returned. Unless explicitly specified,
// this function does not allow the leaf component of path to be a symbolic link
// (though intermediate components of the path can be symbolic links and will be
// resolved in the resolution of the path), and an error will be returned if
// this is the case (though on POSIX systems it will not be
// ErrUnsupportedOpenType). However, if allowSymbolicLinkLeaf is true, then this
// function will allow resolution of a path leaf component that's a symbolic
// link. In this case, the referenced object must still be a directory or
// regular file, and the returned object will still be either a Directory or
// ReadableFile.
func Open(path string, allowSymbolicLinkLeaf bool) (io.Closer, *Metadata, error) {
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
	// problem since it doens't support AT_SYMLINK_NOFOLLOW. Essentially,
	// there's no way to "open" a symbolic link - it can only be read with
	// readlink and its ilk. Since ELOOP still sort of makes sense (we've
	// encountered too many symbolic links at the path leaf), we return it
	// unmodified.
	//
	// HACK: We use the same looping construct as Go to avoid golang/go#11180.
	flags := unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_CLOEXEC
	if allowSymbolicLinkLeaf {
		flags &^= unix.O_NOFOLLOW
	}
	var descriptor int
	for {
		if d, err := unix.Open(path, flags, 0); err == nil {
			descriptor = d
			break
		} else if runtime.GOOS == "darwin" && err == unix.EINTR {
			continue
		} else {
			return nil, nil, err
		}
	}

	// Grab metadata for the file.
	var rawMetadata unix.Stat_t
	if err := unix.Fstat(descriptor, &rawMetadata); err != nil {
		unix.Close(descriptor)
		return nil, nil, errors.Wrap(err, "unable to query file metadata")
	}

	// Extract modification time specification.
	modificationTime := extractModificationTime(&rawMetadata)

	// Convert the raw system-level metadata.
	metadata := &Metadata{
		Name:             filepath.Base(path),
		Mode:             Mode(rawMetadata.Mode),
		Size:             uint64(rawMetadata.Size),
		ModificationTime: time.Unix(modificationTime.Unix()),
		DeviceID:         uint64(rawMetadata.Dev),
		FileID:           uint64(rawMetadata.Ino),
	}

	// Wrap the descriptor up in an os.File object.
	file := os.NewFile(uintptr(descriptor), path)

	// Dispatch further construction according to type.
	switch metadata.Mode & ModeTypeMask {
	case ModeTypeDirectory:
		return &Directory{
			descriptor: descriptor,
			file:       file,
		}, metadata, nil
	case ModeTypeFile:
		return file, metadata, nil
	default:
		unix.Close(descriptor)
		return nil, nil, ErrUnsupportedOpenType
	}
}
