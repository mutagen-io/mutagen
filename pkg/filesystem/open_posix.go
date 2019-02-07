// +build !windows

package filesystem

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
)

// Open opens a filesystem path for traversal and operations. It will return
// either a Directory or ReadableFile object, along with Metadata that can be
// used to determine the type of object being returned.
func Open(path string) (interface{}, *Metadata, error) {
	// Open the file using the os package's infrastructure. We specify
	// O_NOFOLLOW to enforce that the underlying open operation doesn't follow
	// symbolic links at the path leaf (intermedaite symbolic links are still
	// allowed).
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
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, nil, err
	}

	// Grab file metadata.
	fileMetadata, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, errors.Wrap(err, "unable to query file metadata")
	}

	// Extract the raw system-level metadata.
	rawMetadata, ok := fileMetadata.Sys().(*syscall.Stat_t)
	if !ok {
		file.Close()
		return nil, nil, errors.New("unable to extract raw file metadata")
	}

	// Convert the raw system-level metadata.
	metadata := &Metadata{
		Name:             filepath.Base(path),
		Mode:             Mode(rawMetadata.Mode),
		Size:             uint64(rawMetadata.Size),
		ModificationTime: fileMetadata.ModTime(),
		DeviceID:         uint64(rawMetadata.Dev),
		FileID:           uint64(rawMetadata.Ino),
	}

	// Dispatch further construction according to type.
	switch metadata.Mode & ModeTypeMask {
	case ModeTypeDirectory:
		return &Directory{
			file:       file,
			descriptor: int(file.Fd()),
		}, metadata, nil
	case ModeTypeFile:
		return file, metadata, nil
	default:
		file.Close()
		return nil, nil, ErrUnsupportedRootType
	}
}
