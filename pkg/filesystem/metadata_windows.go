package filesystem

import (
	"errors"
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/mutagen-io/mutagen/pkg/filesystem/internal/syscall"
)

// queryHandleMetadata performs a metadata query using a Windows file handle. It
// must be passed the base name of the path used to open the handle. It supports
// files, directories, and symbolic links. It behavior designed to match that of
// the os.File.Stat method on Windows, specifically the unexported
// newFileStatFromGetFileInformationByHandle function and the various methods of
// os.fileStat. The behavior of this function, in particular its classification
// of file types in modes, must match that of the Go standard library in order
// for this package to function correctly (because the Windows implementation of
// this package partially relies on the standard os package and certain
// invariants will break if this classification behavior differs).
func queryHandleMetadata(name string, handle windows.Handle) (*Metadata, error) {
	// Query the file type to ensure that it's an on-disk type (i.e. a file,
	// directory, or symbolic link).
	if t, err := windows.GetFileType(handle); err != nil {
		return nil, fmt.Errorf("unable to determine file type: %w", err)
	} else if t != windows.FILE_TYPE_DISK {
		return nil, errors.New("handle does not refer to on-disk type")
	}

	// Perform a general file metadata query.
	var metadata windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &metadata); err != nil {
		return nil, fmt.Errorf("unable to query file metadata: %w", err)
	}

	// If the handle refers to a reparse point, then determine whether or not
	// it's a symbolic link. When dealing with reparse points, we need to
	// perform an attribute query. Unfortunately this query isn't supported on
	// some (or perhaps any) non-NTFS filesystems, so we can't use it as our
	// only query (even though it returns the same FileAttributes field as the
	// general query above). In cases where this query returns an invalid
	// parameter error, we assume that we're on a non-NTFS filesystem, in which
	// case symbolic links aren't supported in any case. See golang/go#29214 for
	// more information.
	var symbolicLink bool
	if metadata.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		var attributes syscall.FileAttributeTagInfo
		if err := windows.GetFileInformationByHandleEx(
			handle,
			windows.FileAttributeTagInfo,
			(*byte)(unsafe.Pointer(&attributes)),
			uint32(unsafe.Sizeof(attributes)),
		); err != nil {
			if err != windows.ERROR_INVALID_PARAMETER {
				return nil, fmt.Errorf("unable to query reparse point attributes: %w", err)
			}
		} else {
			// Determine whether or not we're dealing with a symbolic link. This
			// logic follows that in os.fileStat.isSymlink.
			//
			// TODO: Update this definition once golang/go#42184 is resolved.
			symbolicLink = attributes.ReparseTag == windows.IO_REPARSE_TAG_SYMLINK ||
				attributes.ReparseTag == windows.IO_REPARSE_TAG_MOUNT_POINT
		}
	}

	// Compute the mode. Note that the logic here needs to match that of
	// os.fileStat.Mode.
	mode := Mode(0666)
	if metadata.FileAttributes&windows.FILE_ATTRIBUTE_READONLY != 0 {
		mode = Mode(0444)
	}
	if symbolicLink {
		mode |= ModeTypeSymbolicLink
	} else if metadata.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0 {
		mode |= ModeTypeDirectory | 0111
	}

	// Compute the size.
	size := uint64(metadata.FileSizeHigh)<<32 + uint64(metadata.FileSizeLow)

	// Compute the modification time.
	modificationTime := time.Unix(0, metadata.LastWriteTime.Nanoseconds())

	// Success.
	return &Metadata{
		Name:             name,
		Mode:             mode,
		Size:             size,
		ModificationTime: modificationTime,
	}, nil
}
