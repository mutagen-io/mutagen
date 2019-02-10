package sync

import (
	"github.com/pkg/errors"

	fs "github.com/havoc-io/mutagen/pkg/filesystem"
)

const (
	// allReadWritePermissionMask is a union of the user, group, and others read
	// and write permission bits.
	allReadWritePermissionMask = fs.ModePermissionUserRead | fs.ModePermissionUserWrite |
		fs.ModePermissionGroupRead | fs.ModePermissionGroupWrite |
		fs.ModePermissionOthersRead | fs.ModePermissionOthersWrite

	// allExecutePermissionMask is a union of the user, group, and others
	// executable permission bits.
	allExecutePermissionMask = fs.ModePermissionUserExecute |
		fs.ModePermissionGroupExecute |
		fs.ModePermissionOthersExecute
)

// EnsureDefaultFileModeValid validates that a user-provided default file mode
// is valid in the context of "portable" permission propagation. In particular,
// it enforces that the mode is non-0 and that no executable bits are set (since
// these should be regulated by the synchronization algorithm).
func EnsureDefaultFileModeValid(mode fs.Mode) error {
	// Verify that only valid bits are set. Executability is excluded since it
	// is controlled by Mutagen.
	if (mode & allReadWritePermissionMask) != mode {
		return errors.New("executability bits detected in file mode")
	}

	// Verify that the mode is non-zero. This should never be the case, because
	// we treat a zero-value mode as unspecified.
	if mode == 0 {
		return errors.New("zero-value file permission mode specified")
	}

	// Success.
	return nil
}

// EnsureDefaultDirectoryModeValid validates that a user-provided default
// directory mode is valid in the context of Mutagen's synchronization
// algorithms.
func EnsureDefaultDirectoryModeValid(mode fs.Mode) error {
	// Verify that only base permissions are set.
	if (mode & fs.ModePermissionsMask) != mode {
		return errors.New("non-permission bits detected in directory mode")
	}

	// Verify that the mode is non-zero. This should never be the case, because
	// we treat a zero-value mode as unspecified.
	if mode == 0 {
		return errors.New("zero-value directory permission mode specified")
	}

	// Success.
	return nil
}

// anyExecutableBitSet returns true if any executable bit is set on the file,
// false otherwise.
func anyExecutableBitSet(mode fs.Mode) bool {
	return (mode & allExecutePermissionMask) != 0
}

// markExecutableForReaders sets the executable bit for the mode for any case
// where the corresponding read bit is set.
func markExecutableForReaders(mode fs.Mode) fs.Mode {
	// Set the user executable bit if necessary.
	if (mode & fs.ModePermissionUserRead) != 0 {
		mode |= fs.ModePermissionUserExecute
	}

	// Set the group executable bit if necessary.
	if (mode & fs.ModePermissionGroupRead) != 0 {
		mode |= fs.ModePermissionGroupExecute
	}

	// Set the others executable bit if necessary.
	if (mode & fs.ModePermissionOthersRead) != 0 {
		mode |= fs.ModePermissionOthersExecute
	}

	// Done.
	return mode
}
