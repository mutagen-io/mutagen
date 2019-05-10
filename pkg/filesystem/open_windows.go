package filesystem

import (
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"golang.org/x/sys/windows"

	osvendor "github.com/havoc-io/mutagen/pkg/filesystem/internal/third_party/os"
)

// Open opens a filesystem path for traversal and operations. It will return
// either a Directory or ReadableFile object (as an io.Closer for convenient
// closing access without casting), along with Metadata that can be used to
// determine the type of object being returned. Unless explicitly specified,
// this function does not allow the leaf component of path to be a symbolic link
// (though intermediate components of the path can be symbolic links and will be
// resolved in the resolution of the path), and an error will be returned if
// this is the case. However, if allowSymbolicLinkLeaf is true, then this
// function will allow resolution of a path leaf component that's a symbolic
// link. In this case, the referenced object must still be a directory or
// regular file, and the returned object will still be either a Directory or
// ReadableFile.
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
		return nil, nil, errors.Wrap(err, "unable to convert path to UTF-16")
	}

	// Open the path in a manner that is suitable for reading, doesn't allow for
	// other threads or processes to delete or rename the file while open,
	// avoids symbolic link traversal (at the path leaf), and has suitable
	// semantics for both files and directories.
	flags := uint32(windows.FILE_ATTRIBUTE_NORMAL | windows.FILE_FLAG_BACKUP_SEMANTICS | windows.FILE_FLAG_OPEN_REPARSE_POINT)
	if allowSymbolicLinkLeaf {
		flags = uint32(windows.FILE_ATTRIBUTE_NORMAL | windows.FILE_FLAG_BACKUP_SEMANTICS)
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
		return nil, nil, errors.Wrap(err, "unable to open path")
	}

	// Query raw file metadata.
	var rawMetadata windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &rawMetadata); err != nil {
		windows.CloseHandle(handle)
		return nil, nil, errors.Wrap(err, "unable to query file metadata")
	}

	// Verify that the handle does not represent a symbolic link. Even if we
	// allow symbolic links in the leaf position of the path, we should not end
	// up with one (it should resolve).
	//
	// Note that FILE_ATTRIBUTE_REPARSE_POINT can be or'd with
	// FILE_ATTRIBUTE_DIRECTORY (since symbolic links are "typed" on Windows),
	// so we have to explicitly exclude reparse points before checking types.
	//
	// TODO: Are there additional attributes upon which we should reject here?
	// The Go os.File implementation doesn't seem to for normal os.Open
	// operations, so I guess we don't need to either, but we should keep the
	// option in mind.
	if rawMetadata.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		windows.CloseHandle(handle)
		return nil, nil, ErrUnsupportedOpenType
	}

	// Determine whether or not this is a directory.
	isDirectory := rawMetadata.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0

	// Handle os.File creation based on type.
	var file *os.File
	if isDirectory {
		file, err = os.Open(path)
		if err != nil {
			windows.CloseHandle(handle)
			return nil, nil, errors.Wrap(err, "unable to open file object for directory")
		}
	} else {
		file = os.NewFile(uintptr(handle), path)
	}

	// Grab pre-converted metadata via the os.File object. For directories, this
	// is path-based under the hood, but it's fine since we're already holding
	// the directory handle open.
	fileMetadata, err := file.Stat()
	if err != nil {
		if isDirectory {
			windows.CloseHandle(handle)
		}
		file.Close()
		return nil, nil, errors.Wrap(err, "unable to query file metadata")
	}

	// Convert metadata.
	metadata := &Metadata{
		Name:             fileMetadata.Name(),
		Mode:             Mode(fileMetadata.Mode()),
		Size:             uint64(fileMetadata.Size()),
		ModificationTime: fileMetadata.ModTime(),
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
