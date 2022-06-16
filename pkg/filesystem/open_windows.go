package filesystem

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"

	osvendor "github.com/mutagen-io/mutagen/pkg/filesystem/internal/third_party/os"
)

// Open opens a filesystem path for traversal and/or other operations. It will
// return either a Directory or an io.ReadSeekCloser object (as an io.Closer for
// convenient closing access without casting), along with Metadata that can be
// used to determine the type of object being returned. Unless requested, this
// function does not allow the leaf component of path to be a symbolic link
// (though intermediate components of the path can be symbolic links and will be
// resolved in the resolution of the path), and an error will be returned if
// this is the case. However, if allowSymbolicLinkLeaf is true, then this
// function will allow resolution of a path leaf component that's a symbolic
// link. In this case, the referenced object must still be a directory or
// regular file, and the returned object will still be either a Directory or an
// io.ReadSeekCloser.
func Open(path string, allowSymbolicLinkLeaf bool) (io.Closer, *Metadata, error) {
	// Verify that the provided path is absolute. This is a requirement on
	// Windows, where all of our operations are path-based.
	if !filepath.IsAbs(path) {
		return nil, nil, errors.New("path is not absolute")
	}

	// Fix long paths.
	path = osvendor.FixLongPath(path)

	// Convert the path to UTF-16.
	path16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to convert path to UTF-16: %w", err)
	}

	// Open the path in a manner that is suitable for reading, doesn't allow for
	// other threads or processes to delete or rename the file while open,
	// avoids symbolic link traversal (at the path leaf), and has suitable
	// semantics for both files and directories.
	flags := uint32(windows.FILE_ATTRIBUTE_NORMAL | windows.FILE_FLAG_BACKUP_SEMANTICS)
	if !allowSymbolicLinkLeaf {
		flags |= windows.FILE_FLAG_OPEN_REPARSE_POINT
	}
	handle, err := windows.CreateFile(
		path16,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		flags,
		0,
	)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, err
		}
		return nil, nil, fmt.Errorf("unable to open path: %w", err)
	}

	// Query handle metadata.
	metadata, err := queryHandleMetadata(filepath.Base(path), handle)
	if err != nil {
		windows.CloseHandle(handle)
		return nil, nil, fmt.Errorf("unable to query file handle metadata: %w", err)
	}

	// Verify that we're not dealing with a symbolic link. If we are allowing
	// symbolic links, then they should have been resolved by CreateFile.
	if metadata.Mode&ModeTypeSymbolicLink != 0 {
		windows.CloseHandle(handle)
		return nil, nil, ErrUnsupportedOpenType
	}

	// Handle os.File creation based on type.
	var file *os.File
	isDirectory := metadata.Mode&ModeTypeDirectory != 0
	if isDirectory {
		file, err = os.Open(path)
		if err != nil {
			windows.CloseHandle(handle)
			return nil, nil, fmt.Errorf("unable to open file object for directory: %w", err)
		}
	} else {
		file = os.NewFile(uintptr(handle), path)
	}

	// Success.
	if isDirectory {
		return &Directory{
			handle: handle,
			file:   file,
		}, metadata, nil
	} else {
		return file, metadata, nil
	}
}
