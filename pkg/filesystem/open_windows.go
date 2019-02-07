package filesystem

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"golang.org/x/sys/windows"
)

// Open opens a filesystem path for traversal and operations. It will return
// either a Directory or ReadableFile object, along with Metadata that can be
// used to determine the type of object being returned.
func Open(path string) (interface{}, *Metadata, error) {
	// Verify that the provided path is absolute. This is a requirement on
	// Windows, where all of our operations are path-based.
	if !filepath.IsAbs(path) {
		return nil, nil, errors.New("path is not absolute")
	}

	// Convert the path to UTF-16.
	path16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to convert path to UTF-16")
	}

	// Open the path in a manner that is suitable for reading, doesn't allow for
	// other threads or processes to delete or rename the file while open,
	// avoids symbolic link traversal (at the path leaf), and has suitable
	// semantics for both files and directories.
	handle, err := windows.CreateFile(
		path16,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to open path")
	}

	// Query raw file metadata.
	var rawMetadata windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &rawMetadata); err != nil {
		windows.CloseHandle(handle)
		return nil, nil, errors.Wrap(err, "unable to query file metadata")
	}

	// Verify that the handle does not represent a symbolic link. Note that
	// FILE_ATTRIBUTE_REPARSE_POINT can be or'd with FILE_ATTRIBUTE_DIRECTORY
	// (since symbolic links are "typed" on Windows), so we have to explicitly
	// exclude reparse points before checking types.
	//
	// TODO: Are there additional attributes upon which we should reject here?
	// The Go os.File implementation doesn't seem to for normal os.Open
	// operations, so I guess we don't need to either, but we should keep the
	// option in mind.
	if rawMetadata.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		windows.CloseHandle(handle)
		return nil, nil, ErrUnsupportedRootType
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
