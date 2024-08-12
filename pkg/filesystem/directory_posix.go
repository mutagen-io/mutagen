//go:build !windows

package filesystem

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"golang.org/x/sys/unix"

	"github.com/mutagen-io/mutagen/pkg/state"
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
	// exhausted indicates that the directory's contents have been read and that
	// a seek is required before reading them again.
	exhausted bool
	// renameatNoReplaceUnsupported is marked if
	// renameatNoReplaceRetryingOnEINTR is found to be unsupported with this
	// directory as a target.
	renameatNoReplaceUnsupported state.Marker

	logger *logging.Logger
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
	return mkdiratRetryingOnEINTR(d.descriptor, name, 0700)
}

// createTemporaryFilePRNGLock serializes access to createTemporaryFilePRNG.
var createTemporaryFilePRNGLock sync.Mutex

// createTemporaryFilePRNG provides pseudorandom numbers for filenames in
// Directory.CreateTemporaryFile.
var createTemporaryFilePRNG *rand.Rand

func init() {
	// Read random data to compute a seed for the pseudorandom number generator.
	var seedBytes [8]byte
	if _, err := cryptorand.Read(seedBytes[:]); err != nil {
		panic("unable to read random bytes for seed")
	}

	// Initialize the pseudorandom number generator.
	createTemporaryFilePRNG = rand.New(rand.NewSource(int64(binary.BigEndian.Uint64(seedBytes[:]))))
}

// CreateTemporaryFile creates a new temporary file using the specified name
// pattern inside the directory. Pattern behavior follows that of os.CreateTemp.
// The file will be created with user-only read/write permissions.
func (d *Directory) CreateTemporaryFile(pattern string) (string, io.WriteCloser, error) {
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

	// Iterate until we can find a free file name.
	try := 0
	for {
		// Compute the next potential name using a pseudorandom component.
		createTemporaryFilePRNGLock.Lock()
		random := createTemporaryFilePRNG.Int()
		createTemporaryFilePRNGLock.Unlock()
		name := prefix + strconv.Itoa(random) + suffix

		// Open the file. Note that we needn't specify O_NOFOLLOW here since
		// we're enforcing that the file doesn't already exist.
		descriptor, err := openatRetryingOnEINTR(d.descriptor, name, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC, 0600)
		if err != nil {
			if os.IsExist(err) {
				if try++; try < 10000 {
					continue
				}
				return "", nil, errors.New("exhausted potential file names")
			}
			return "", nil, err
		}

		// Wrap up the descriptor in a file object.
		file := os.NewFile(uintptr(descriptor), name)

		// Success.
		return name, file, nil
	}
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
	return symlinkatRetryingOnEINTR(target, d.descriptor, name)
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
		if err := fchownatRetryingOnEINTR(d.descriptor, name, ownership.ownerID, ownership.groupID, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return fmt.Errorf("unable to set ownership information: %w", err)
		}
	}

	// Set permissions, if specified.
	//
	// HACK: On Linux, the AT_SYMLINK_NOFOLLOW flag is not supported by fchmodat
	// and will result in an ENOTSUP error, so we have to use a workaround that
	// opens a file and then uses fchmod in order to avoid setting permissions
	// across a symbolic link.
	mode &= ModePermissionsMask
	if mode != 0 {
		if runtime.GOOS == "linux" {
			if f, err := openatRetryingOnEINTR(d.descriptor, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0); err != nil {
				return fmt.Errorf("unable to open file: %w", err)
			} else if err = fchmodRetryingOnEINTR(f, uint32(mode)); err != nil {
				mustCloseConsideringEINTR(f, d.logger)
				return fmt.Errorf("unable to set permission bits on file: %w", err)
			} else if err = closeConsideringEINTR(f); err != nil {
				return fmt.Errorf("unable to close file: %w", err)
			}
		} else {
			if err := fchmodatRetryingOnEINTR(d.descriptor, name, uint32(mode), unix.AT_SYMLINK_NOFOLLOW); err != nil {
				return fmt.Errorf("unable to set permission bits: %w", err)
			}
		}
	}

	// Success.
	return nil
}

// open is the underlying open implementation shared by OpenDirectory and
// OpenFile. It returns the file descriptor corresponding to the target, the
// target metadata if the target is a file (nil otherwise), or any error.
func (d *Directory) open(name string, wantDirectory bool) (int, *Metadata, error) {
	// Verify that the name is valid.
	if wantDirectory && name == "." {
		// As a special case, we allow directories to be re-opened on POSIX
		// systems. This is safe since it doesn't allow traversal.
	} else if err := ensureValidName(name); err != nil {
		return -1, nil, err
	}

	// Open the file for reading while avoiding symbolic link traversal. If a
	// directory has been requested, then enforce its type here.
	flags := unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_CLOEXEC | extraOpenFlags
	if wantDirectory {
		flags |= unix.O_DIRECTORY
	}
	descriptor, err := openatRetryingOnEINTR(d.descriptor, name, flags, 0)
	if err != nil {
		return -1, nil, err
	}

	// If a file has been requested, then verify that's what we've received.
	// This (along with the directory enforcement above) keeps parity with the
	// Windows implementation, where checking file type is required for the
	// implementation to work at all. Unfortunately there's no O_FILE flag
	// analagous to O_DIRECTORY that we can use with openat, so we have to check
	// this using an fstat operation. There is some overhead to performing this
	// check, of course, and on POSIX we could probably live without it (simply
	// allowing other methods on the resulting object to fail due to an
	// unexpected type), but given the typical filesystem access patterns at
	// play when using this code, the overhead will be minimal since this
	// information should still be in the OS's stat cache.
	var metadata *Metadata
	if !wantDirectory {
		var rawMetadata unix.Stat_t
		if err := fstatRetryingOnEINTR(descriptor, &rawMetadata); err != nil {
			mustCloseConsideringEINTR(descriptor, d.logger)
			return -1, nil, fmt.Errorf("unable to query file metadata: %w", err)
		} else if Mode(rawMetadata.Mode)&ModeTypeMask != ModeTypeFile {
			mustCloseConsideringEINTR(descriptor, d.logger)
			return -1, nil, errors.New("path is not a file")
		}
		metadata = &Metadata{
			Name:             name,
			Mode:             Mode(rawMetadata.Mode),
			Size:             uint64(rawMetadata.Size),
			ModificationTime: time.Unix(rawMetadata.Mtim.Unix()),
			DeviceID:         uint64(rawMetadata.Dev),
			FileID:           uint64(rawMetadata.Ino),
		}
	}

	// Success.
	return descriptor, metadata, nil
}

// OpenDirectory opens the directory within the directory specified by name. On
// POSIX systems, the directory itself can be re-opened (with a different
// underlying file handle pointing to the same directory) by passing "." to this
// function.
func (d *Directory) OpenDirectory(name string, logger *logging.Logger) (*Directory, error) {
	// Call the underlying open method.
	descriptor, _, err := d.open(name, true)
	if err != nil {
		return nil, err
	}

	// Success.
	return &Directory{
		descriptor: descriptor,
		file:       os.NewFile(uintptr(descriptor), name),
		logger:     logger,
	}, nil
}

// ReadContentNames queries the directory contents and returns their base names.
// It does not return "." or ".." entries.
func (d *Directory) ReadContentNames() ([]string, error) {
	// If we've already performed a read on the directory's contents, then we
	// need to rewind the directory before performing another read.
	if d.exhausted {
		if offset, err := seekConsideringEINTR(d.descriptor, 0, 0); err != nil {
			return nil, fmt.Errorf("unable to reset directory read pointer: %w", err)
		} else if offset != 0 {
			return nil, errors.New("directory offset is non-zero after seek operation")
		}
		d.exhausted = false
	}

	// Read content names. Fortunately we can use the os.File implementation for
	// this since it operates on the underlying file descriptor directly. We
	// always mark the directory as exhausted, even if we fail to read it all
	// the way to the end.
	names, err := d.file.Readdirnames(0)
	d.exhausted = true
	if err != nil {
		return nil, err
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

	// Perform the actual query operation.
	return d.readContentMetadata(name)
}

// readContentMetadata reads metadata for the content within the directory
// specified by name, but does not perform a check for name validity.
func (d *Directory) readContentMetadata(name string) (*Metadata, error) {
	// Query metadata.
	var metadata unix.Stat_t
	if err := fstatatRetryingOnEINTR(d.descriptor, name, &metadata, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return nil, err
	}

	// Success.
	return &Metadata{
		Name:             name,
		Mode:             Mode(metadata.Mode),
		Size:             uint64(metadata.Size),
		ModificationTime: time.Unix(metadata.Mtim.Unix()),
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
		return nil, fmt.Errorf("unable to read directory content names: %w", err)
	}

	// Allocate the result slice with enough capacity to accommodate all
	// entries.
	results := make([]*Metadata, 0, len(names))

	// Loop over names and grab their individual metadata.
	for _, name := range names {
		// Grab metadata for this entry. We don't need to validate its name in
		// this scenario since we just received it from the OS. If the file has
		// disappeared between listing and the metadata query, then just pretend
		// that it never existed.
		if m, err := d.readContentMetadata(name); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("unable to access content metadata: %w", err)
		} else {
			results = append(results, m)
		}
	}

	// Success.
	return results, nil
}

// OpenFile opens the file within the directory specified by name.
func (d *Directory) OpenFile(name string) (io.ReadSeekCloser, *Metadata, error) {
	// Perform the open operation.
	descriptor, metadata, err := d.open(name, false)
	if err != nil {
		return nil, nil, err
	}

	// Convert the file descriptor to a usable type.
	return file(descriptor), metadata, err
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
	// symbolic link and with buffer space to spare. This is the only way to
	// approach the problem because readlink and its ilk don't provide any
	// mechanism for determining the untruncated length of the symbolic link.
	for size := readlinkInitialBufferSize; ; size *= 2 {
		// Allocate a buffer.
		buffer := make([]byte, size)

		// Read the symbolic link target.
		count, err := readlinkatRetryingOnEINTR(d.descriptor, name, buffer)

		// Handle errors. If we see ERANGE on AIX systems, it's an indication
		// that the buffer size is too small.
		if runtime.GOOS == "aix" && err == unix.ERANGE {
			continue
		} else if err != nil {
			return "", err
		}

		// Verify that the count is sane. We diverge from the os.Readlink
		// implementation here (which just sets this value to 0 if negative),
		// because POSIX specifically says a return value of -1 is indicative of
		// an error.
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
	return unlinkatRetryingOnEINTR(d.descriptor, name, unix.AT_REMOVEDIR)
}

// RemoveFile deletes a file with the specified name inside the directory.
func (d *Directory) RemoveFile(name string) error {
	// Verify that the name is valid.
	if err := ensureValidName(name); err != nil {
		return err
	}

	// Remove the file.
	return unlinkatRetryingOnEINTR(d.descriptor, name, 0)
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
	replace bool,
) error {
	// If a source directory has been provided, then verify that the source name
	// is valid and extract the source directory descriptor.
	sourceDescriptor := unix.AT_FDCWD
	if sourceDirectory != nil {
		if err := ensureValidName(sourceNameOrPath); err != nil {
			return fmt.Errorf("source name invalid: %w", err)
		}
		sourceDescriptor = sourceDirectory.descriptor
	}

	// If a target directory has been provided, then verify that the target name
	// is valid and extract the target directory descriptor.
	targetDescriptor := unix.AT_FDCWD
	if targetDirectory != nil {
		if err := ensureValidName(targetNameOrPath); err != nil {
			return fmt.Errorf("target name invalid: %w", err)
		}
		targetDescriptor = targetDirectory.descriptor
	}

	// If we're allowing the target to be replaced, then just attempt a standard
	// rename operation.
	if replace {
		return renameatRetryingOnEINTR(
			sourceDescriptor, sourceNameOrPath,
			targetDescriptor, targetNameOrPath,
		)
	}

	// Since we're not allowing replacement, we need to ensure that the target
	// doesn't exist. Some platforms provide specialized renameat variants and
	// flags for this purpose, so we'll see if that's the case first. We'll skip
	// this if we've already determined that the target directory's filesystem
	// doesn't support this mechanism.
	if targetDirectory == nil || !targetDirectory.renameatNoReplaceUnsupported.Marked() {
		err := renameatNoReplaceRetryingOnEINTR(
			sourceDescriptor, sourceNameOrPath,
			targetDescriptor, targetNameOrPath,
		)
		if err == nil || (err != unix.ENOTSUP && err != unix.ENOSYS) {
			return err
		} else if err == unix.ENOTSUP && targetDirectory != nil {
			targetDirectory.renameatNoReplaceUnsupported.Mark()
		}
	}

	// There either isn't a non-replacing variant of renameat available or it
	// isn't supported on this platform or target filesystem. In any case, we're
	// falling back to the slower and less atomic method, so check if the target
	// exists.
	var probeErr error
	if targetDirectory != nil {
		_, probeErr = targetDirectory.ReadContentMetadata(targetNameOrPath)
	} else {
		_, probeErr = os.Lstat(targetNameOrPath)
	}
	if probeErr == nil {
		return os.ErrExist
	} else if !os.IsNotExist(probeErr) {
		return fmt.Errorf("unable to probe target existence: %w", probeErr)
	}

	// RACE: There's a race window here between the time of our check and the
	// time that the file is renamed. This is a limitation of the POSIX API.

	// Attempt the rename operation.
	return renameatRetryingOnEINTR(
		sourceDescriptor, sourceNameOrPath,
		targetDescriptor, targetNameOrPath,
	)
}

// IsCrossDeviceError checks whether or not an error returned from rename
// represents a cross-device error.
func IsCrossDeviceError(err error) bool {
	return err == unix.EXDEV
}
