// +build !windows

package filesystem

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"golang.org/x/sys/unix"

	fssyscall "github.com/havoc-io/mutagen/pkg/filesystem/internal/syscall"
)

// ensureValidName verifies that the provided name does not reference the
// current directory, the parent directory, or contain a path separator
// character.
func ensureValidName(name string) error {
	// Verify that the name does not reference the directory itself or the
	// parent directory.
	if name == "." {
		return errors.New("name is directory reference")
	} else if name == ".." {
		return errors.New("name is parent directory reference")
	}

	// Verify that the path separator character does not appear in the name.
	if strings.IndexByte(name, os.PathSeparator) != -1 {
		return errors.New("path separator appears in name")
	}

	// Success.
	return nil
}

// Directory represents a directory on disk and provides race-free operations on
// the directory's contents. All of its operations avoid the traversal of
// symbolic links.
type Directory struct {
	// descriptor is the file descriptor for the directory, designed to be used
	// in conjunction with POSIX *at functions. It is wrapped by the os.File
	// object below (file) and should not be closed directly.
	descriptor int
	// file is an os.File object which wraps the directory descriptor. It is
	// required for its Readdirnames function, since there's no other portable
	// way to do this from Go.
	file *os.File
}

// Close closes the directory.
func (d *Directory) Close() error {
	return d.file.Close()
}

// Descriptor provides access to the raw file descriptor underlying the
// directory. It should not be used or retained beyond the point in time where
// the Close method is called, and it should not be closed externally. Its
// usefulness is to code which relies on file-descriptor-based operations. This
// method does not exist on Windows systems, so it should only be used in
// POSIX-specific code.
func (d *Directory) Descriptor() int {
	return d.descriptor
}

// CreateDirectory creates a new directory with the specified name inside the
// directory. The directory will be created with user-only read/write/execute
// permissions.
func (d *Directory) CreateDirectory(name string) error {
	// Verify that the name is valid.
	if err := ensureValidName(name); err != nil {
		return err
	}

	// Create the directory.
	return unix.Mkdirat(d.descriptor, name, 0700)
}

// maximumTemporaryFileRetries is the maximum number of retries that we'll
// perform when trying to find an available temporary file name.
const maximumTemporaryFileRetries = 256

// CreateTemporaryFile creates a new temporary file using the specified name
// pattern inside the directory. Pattern behavior follows that of
// io/ioutil.TempFile. The file will be created with user-only read/write
// permissions.
func (d *Directory) CreateTemporaryFile(pattern string) (string, WritableFile, error) {
	// Verify that the name is valid. This should still be a sensible operation
	// for pattern specifications.
	if err := ensureValidName(pattern); err != nil {
		return "", nil, err
	}

	// Parse the pattern into prefix and suffix components.
	var prefix, suffix string
	if starIndex := strings.LastIndex(pattern, "*"); starIndex != -1 {
		prefix, suffix = pattern[:starIndex], pattern[starIndex+1:]
	} else {
		prefix = pattern
	}

	// Iterate until we can find a free file name. We take a slightly simpler
	// approach than the io/ioutil.TempFile implementation and skip the user of
	// a random number generator.
	// TODO: Is it worth going through the trouble of using a random number
	// generator for this?
	for i := 0; i < maximumTemporaryFileRetries; i++ {
		// Compute the next potential name.
		name := prefix + strconv.Itoa(i) + suffix

		// Attempt to open the file. Note that we needn't specify O_NOFOLLOW
		// here since we're enforcing that the file doesn't already exist.
		descriptor, err := unix.Openat(d.descriptor, name, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC, 0600)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return "", nil, errors.Wrap(err, "unable to create file")
		}

		// Wrap up the descriptor in a file object.
		file := os.NewFile(uintptr(descriptor), name)

		// Success.
		return name, file, nil
	}

	// At this point, we've exhausted our maximum number of retries.
	return "", nil, errors.New("exhausted potential file names")
}

// CreateSymbolicLink creates a new symbolic link with the specified name and
// target inside the directory. The symbolic link is created with the default
// system permissions (which, generally speaking, don't apply to the symbolic
// link itself).
func (d *Directory) CreateSymbolicLink(name, target string) error {
	// Verify that the name is valid.
	if err := ensureValidName(name); err != nil {
		return err
	}

	// Create the symbolic link.
	return fssyscall.Symlinkat(target, d.descriptor, name)
}

// SetPermissions sets the permissions on the content within the directory
// specified by name. Ownership information is set first, followed by
// permissions extracted from the mode using ModePermissionsMask. Ownership
// setting can be skipped completely by providing a nil OwnershipSpecification
// or a specification with both components unset. An OwnershipSpecification may
// also include only certain components, in which case only those components
// will be set. Permission setting can be skipped by providing a mode value that
// yields 0 after permission bit masking.
func (d *Directory) SetPermissions(name string, ownership *OwnershipSpecification, mode Mode) error {
	// Verify that the name is valid.
	if err := ensureValidName(name); err != nil {
		return err
	}

	// Set ownership information, if specified.
	if ownership != nil && (ownership.ownerID != -1 || ownership.groupID != -1) {
		if err := unix.Fchownat(d.descriptor, name, ownership.ownerID, ownership.groupID, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return errors.Wrap(err, "unable to set ownership information")
		}
	}

	// Set permissions, if specified.
	//
	// HACK: On Linux, the AT_SYMLINK_NOFOLLOW flag is not supported by fchmodat
	// and will result in an ENOTSUP error, so we have to use a workaround that
	// opens a file and then uses fchmod in order to avoid setting permissions
	// across a symbolic link. Fortunately, because we're on Linux, we don't
	// need the looping construct used above to avoid golang/go#11180.
	mode &= ModePermissionsMask
	if mode != 0 {
		if runtime.GOOS == "linux" {
			if f, err := unix.Openat(d.descriptor, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0); err != nil {
				return errors.Wrap(err, "unable to open file")
			} else if err = unix.Fchmod(f, uint32(mode)); err != nil {
				unix.Close(f)
				return errors.Wrap(err, "unable to set permission bits on file")
			} else if err = unix.Close(f); err != nil {
				return errors.Wrap(err, "unable to close file")
			}
		} else {
			if err := unix.Fchmodat(d.descriptor, name, uint32(mode), unix.AT_SYMLINK_NOFOLLOW); err != nil {
				return errors.Wrap(err, "unable to set permission bits")
			}
		}
	}

	// Success.
	return nil
}

// open is the underlying open implementation shared by OpenDirectory and
// OpenFile.
func (d *Directory) open(name string, wantDirectory bool) (int, *os.File, error) {
	// Verify that the name is valid.
	if wantDirectory && name == "." {
		// As a special case, we allow directories to be re-opened on POSIX
		// systems. This is safe since it doesn't allow traversal.
	} else if err := ensureValidName(name); err != nil {
		return -1, nil, err
	}

	// Open the file for reading while avoiding symbolic link traversal. There
	// are a few things to note about the flags that we use. First, we don't
	// specify O_NONBLOCK because that flag applies to the open operation itself
	// rather than the resulting file, and even for the resulting file we don't
	// want to set a non-blocking mode because it isn't useful for directories
	// or regular files. Second, we use the O_CLOEXEC flag to avoid any race
	// conditions with fork/exec infrastructure. It used to be the case that
	// this flag was not supported on every Go platform (and it's still not
	// supported on some of the more esoteric ports (e.g. NaCL and and web
	// platforms)), and there was a race condition between opening files and
	// manually setting close-on-exec behavior, but nowadays all of the "real"
	// POSIX platforms support this flag.
	//
	// HACK: We use the same looping construct as Go to avoid golang/go#11180.
	var descriptor int
	for {
		if d, err := unix.Openat(d.descriptor, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0); err == nil {
			descriptor = d
			break
		} else if runtime.GOOS == "darwin" && err == unix.EINTR {
			continue
		} else {
			return -1, nil, err
		}
	}

	// Verify that we've ended up with the expected file type. This keeps parity
	// with the Windows implementation where checking file type is required for
	// the implementation to work at all. There is some overhead to performing
	// this check, of course, and on POSIX we could live without it (simply
	// allowing other methods on the resulting directory or file object to
	// fail), but given the typical filesystem access patterns at play when
	// using this code (especially in Mutagen), the overhead will be minimal
	// since this information should still be in the OS's stat cache.
	expectedType := ModeTypeFile
	if wantDirectory {
		expectedType = ModeTypeDirectory
	}
	var metadata unix.Stat_t
	if err := unix.Fstat(descriptor, &metadata); err != nil {
		unix.Close(descriptor)
		return -1, nil, errors.Wrap(err, "unable to query file metadata")
	} else if Mode(metadata.Mode)&ModeTypeMask != expectedType {
		unix.Close(descriptor)
		return -1, nil, errors.New("path is not of the expected type")
	}

	// Wrap the descriptor up in a file object. We use the base name for the
	// file since that's the name that was used to "open" the file, which is
	// what os.File.Name is supposed to return (even though we don't expose
	// os.File.Name).
	file := os.NewFile(uintptr(descriptor), name)

	// Success.
	return descriptor, file, nil
}

// OpenDirectory opens the directory within the directory specified by name. On
// POSIX systems, the directory itself can be re-opened (with a different
// underlying file handle pointing to the same directory) by passing "." to this
// function.
func (d *Directory) OpenDirectory(name string) (*Directory, error) {
	// Call the underlying open method.
	descriptor, file, err := d.open(name, true)
	if err != nil {
		return nil, err
	}

	// Success.
	return &Directory{
		descriptor: descriptor,
		file:       file,
	}, nil
}

// ReadContentNames queries the directory contents and returns their base names.
// It does not return "." or ".." entries.
func (d *Directory) ReadContentNames() ([]string, error) {
	// Read content names. Fortunately we can use the os.File implementation for
	// this since it operates on the underlying file descriptor directly.
	names, err := d.file.Readdirnames(0)
	if err != nil {
		return nil, err
	}

	// Seek the directory back to the beginning since the Readdirnames operation
	// will have exhausted its "content".
	if offset, err := unix.Seek(d.descriptor, 0, 0); err != nil {
		return nil, errors.Wrap(err, "unable to reset directory read pointer")
	} else if offset != 0 {
		return nil, errors.New("directory offset is non-zero after seek operation")
	}

	// Filter names (without allocating a new slice).
	results := names[:0]
	for _, name := range names {
		// Watch for names that reference the directory itself or the parent
		// directory. The implementation underlying os.File.Readdirnames does
		// filter these out, but that's not guaranteed by its documentation, so
		// it's better to do this explicitly.
		if name == "." || name == ".." {
			continue
		}

		// Store the name.
		results = append(results, name)
	}

	// Success.
	return names, nil
}

// ReadContentMetadata reads metadata for the content within the directory
// specified by name.
func (d *Directory) ReadContentMetadata(name string) (*Metadata, error) {
	// Verify that the name is valid.
	if err := ensureValidName(name); err != nil {
		return nil, err
	}

	// Query metadata.
	var metadata unix.Stat_t
	if err := unix.Fstatat(d.descriptor, name, &metadata, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return nil, err
	}

	// Extract modification time specification.
	modificationTime := extractModificationTime(&metadata)

	// Success.
	return &Metadata{
		Name:             name,
		Mode:             Mode(metadata.Mode),
		Size:             uint64(metadata.Size),
		ModificationTime: time.Unix(modificationTime.Unix()),
		DeviceID:         uint64(metadata.Dev),
		FileID:           uint64(metadata.Ino),
	}, nil
}

// ReadContents queries the directory contents and their associated metadata.
// While the results of this function can be computed as a combination of
// ReadContentNames and ReadContentMetadata, this function may be significantly
// faster than a naïve combination of the two (e.g. due to the usage of
// FindFirstFile/FindNextFile infrastructure on Windows). This function doesn't
// return metadata for "." or ".." entries.
func (d *Directory) ReadContents() ([]*Metadata, error) {
	// Read content names.
	names, err := d.ReadContentNames()
	if err != nil {
		return nil, errors.Wrap(err, "unable to read directory content names")
	}

	// Allocate the result slice with enough capacity to accommodate all
	// entries.
	results := make([]*Metadata, 0, len(names))

	// Loop over names and grab their individual metadata.
	for _, name := range names {
		// Grab metadata for this entry. If the file has disappeared between
		// listing and the metadata query, then just pretend that it never
		// existed, because from an observability standpoint, it may as well not
		// have.
		if m, err := d.ReadContentMetadata(name); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, errors.Wrap(err, "unable to access content metadata")
		} else {
			results = append(results, m)
		}
	}

	// Success.
	return results, nil
}

// OpenFile opens the file within the directory specified by name.
func (d *Directory) OpenFile(name string) (ReadableFile, error) {
	_, file, err := d.open(name, false)
	return file, err
}

// readlinkInitialBufferSize specifies the initial buffer size to use for
// readlinkat operations. It should be large enough to accommodate most symbolic
// links but not so large that every readlinkat operation incurs an inordinate
// amount of allocation overhead. This value is pinched from the os.Readlink
// implementation.
const readlinkInitialBufferSize = 128

// ReadSymbolicLink reads the target of the symbolic link within the directory
// specified by name.
func (d *Directory) ReadSymbolicLink(name string) (string, error) {
	// Verify that the name is valid.
	if err := ensureValidName(name); err != nil {
		return "", err
	}

	// Loop until we encounter a condition where we successfully read the
	// symbolic link and don't fill our buffer. Unfortunately this is the only
	// way to approach the problem because readlink and its ilk don't provide
	// any mechanism for determining the untruncated length of the symbolic
	// link.
	// TODO: Should we put an upper-bound on this size? The os.Readlink
	// implementation doesn't, and I can't imagine that most OS symbolic link
	// implementations would allow for anything crazy, but maybe it's worth
	// thinking about.
	for size := readlinkInitialBufferSize; ; size *= 2 {
		// Allocate a buffer.
		buffer := make([]byte, size)

		// Attempt to read the symbolic link target.
		// TODO: On AIX systems, readlinkat can specify ERANGE in errno to
		// indicate inadequate buffer space. Watch for this error and continue
		// growing the buffer in that case. See the os.Readlink implementation
		// (for Go 1.12+) for an example of how this is handled.
		count, err := fssyscall.Readlinkat(d.descriptor, name, buffer)
		if err != nil {
			return "", &os.PathError{
				Op:   "readlinkat",
				Path: name,
				Err:  err,
			}
		}

		// Verify that the count is sane.
		if count < 0 {
			return "", errors.New("unknown readlinkat failure occurred")
		}

		// If we've managed to read the target and have buffer space to spare,
		// then we know that we have the full link.
		if count < size {
			return string(buffer[:count]), nil
		}
	}
}

// RemoveDirectory deletes a directory with the specified name inside the
// directory. The removal target must be empty.
func (d *Directory) RemoveDirectory(name string) error {
	// Verify that the name is valid.
	if err := ensureValidName(name); err != nil {
		return err
	}

	// Remove the directory.
	return unix.Unlinkat(d.descriptor, name, fssyscall.AT_REMOVEDIR)
}

// RemoveFile deletes a file with the specified name inside the directory.
func (d *Directory) RemoveFile(name string) error {
	// Verify that the name is valid.
	if err := ensureValidName(name); err != nil {
		return err
	}

	// Remove the file.
	return unix.Unlinkat(d.descriptor, name, 0)
}

// RemoveSymbolicLink deletes a symbolic link with the specified name inside the
// directory.
func (d *Directory) RemoveSymbolicLink(name string) error {
	return d.RemoveFile(name)
}

// Rename performs an atomic rename operation from one filesystem location (the
// source) to another (the target). Each location can be specified in one of two
// ways: either by a combination of directory and (non-path) name or by path
// (with corresponding nil Directory object). Different specification mechanisms
// can be used for each location.
//
// This function does not support cross-device renames. To detect whether or not
// an error is due to an attempted cross-device rename, use the
// IsCrossDeviceError function.
func Rename(
	sourceDirectory *Directory, sourceNameOrPath string,
	targetDirectory *Directory, targetNameOrPath string,
) error {
	// If a source directory has been provided, then verify that the source name
	// is a valid name and not a path.
	if sourceDirectory != nil {
		if err := ensureValidName(sourceNameOrPath); err != nil {
			return errors.Wrap(err, "source name invalid")
		}
	}

	// If a target directory has been provided, then verify that the target name
	// is a valid name and not a path.
	if targetDirectory != nil {
		if err := ensureValidName(targetNameOrPath); err != nil {
			return errors.Wrap(err, "target name invalid")
		}
	}

	// Extract the file descriptors to pass to renameat.
	var sourceDescriptor, targetDescriptor int
	if sourceDirectory != nil {
		sourceDescriptor = sourceDirectory.descriptor
	}
	if targetDirectory != nil {
		targetDescriptor = targetDirectory.descriptor
	}

	// Perform an atomic rename.
	return unix.Renameat(
		sourceDescriptor, sourceNameOrPath,
		targetDescriptor, targetNameOrPath,
	)
}

// IsCrossDeviceError checks whether or not an error returned from rename
// represents a cross-device error.
func IsCrossDeviceError(err error) bool {
	return err == unix.EXDEV
}
