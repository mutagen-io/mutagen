package filesystem

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"

	fssyscall "github.com/mutagen-io/mutagen/pkg/filesystem/internal/syscall"
)

// queryFileHandle performs a metadata query on a Windows file handle, returning
// a pair of results indicating whether or not a file represents a directory and
// whether or not a file represents a symbolic link (note that both can be true
// in the case of directory symbolic links, because Windows symbolic links are
// typed), as well as any error that occurs in making this determination. The
// symbolic link classification behavior of this function must match that of the
// os.fileStat.isSymlink method in the Go standard library in order for this
// package to function correctly (because the Windows implementation of this
// package partially relies on the standard os package and certain invariants
// will break if this classification behavior differs).
func queryFileHandle(handle windows.Handle) (bool, bool, error) {
	// Perform a general file metadata query.
	var metadata windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &metadata); err != nil {
		return false, false, fmt.Errorf("unable to query file metadata: %w", err)
	}

	// Determine whether or not we're dealing with a directory.
	isDirectory := metadata.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0

	// If the file attributes don't indicate a reparse point, then we're done.
	if metadata.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT == 0 {
		return isDirectory, false, nil
	}

	// Since we're dealing with a reparse point, we need to perform an attribute
	// query. Unfortunately this query isn't supported on some (perhaps all)
	// non-NTFS filesystems, so we can't use it as our only query (even though
	// it returns the same FileAttributes field as the general query above). In
	// cases where this query returns an invalid parameter error, we assume that
	// we're on a non-NTFS filesystem, in which case symbolic links aren't
	// supported in any case. See golang/go#29214 for more information.
	var attributes fssyscall.FileAttributeTagInfo
	if err := windows.GetFileInformationByHandleEx(
		handle,
		windows.FileAttributeTagInfo,
		(*byte)(unsafe.Pointer(&attributes)),
		uint32(unsafe.Sizeof(attributes)),
	); err != nil {
		if err == windows.ERROR_INVALID_PARAMETER {
			return isDirectory, false, nil
		}
		return false, false, fmt.Errorf("unable to query reparse point attributes: %w", err)
	}

	// Determine whether or not we're dealing with a symbolic link.
	//
	// TODO: Update this definition once golang/go#42184 is resolved.
	isSymbolicLink := attributes.ReparseTag == windows.IO_REPARSE_TAG_SYMLINK ||
		attributes.ReparseTag == windows.IO_REPARSE_TAG_MOUNT_POINT

	// Success.
	return isDirectory, isSymbolicLink, nil
}
