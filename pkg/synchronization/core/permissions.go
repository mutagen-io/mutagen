package core

import (
	"errors"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

// anyExecutableBitSet returns true if any executable bit is set on the file,
// false otherwise.
func anyExecutableBitSet(mode filesystem.Mode) bool {
	return mode&(filesystem.ModePermissionUserExecute|
		filesystem.ModePermissionGroupExecute|
		filesystem.ModePermissionOthersExecute) != 0
}

// EnsureDefaultFileModeValid validates that a user-provided default file mode
// is valid in the context of "portable" permission propagation. In particular,
// it enforces that the mode is non-0 and that no executable bits are set (since
// these should be regulated by the synchronization algorithm).
func EnsureDefaultFileModeValid(mode filesystem.Mode) error {
	// Verify that the mode is non-zero. This should never be the case, because
	// we treat a zero-value mode as unspecified.
	if mode == 0 {
		return errors.New("zero-value file permission mode specified")
	}

	// Verify that only permission bits are set.
	if (mode & filesystem.ModePermissionsMask) != mode {
		return errors.New("non-permission bits detected in file mode")
	}

	// Verify that no executability bits are set since they're controlled by
	// Mutagen when propagating executability.
	if anyExecutableBitSet(mode) {
		return errors.New("executability bits detected in file mode")
	}

	// Success.
	return nil
}

// EnsureDefaultDirectoryModeValid validates that a user-provided default
// directory mode is valid in the context of Mutagen's synchronization
// algorithms.
func EnsureDefaultDirectoryModeValid(mode filesystem.Mode) error {
	// Verify that the mode is non-zero. This should never be the case, because
	// we treat a zero-value mode as unspecified.
	if mode == 0 {
		return errors.New("zero-value directory permission mode specified")
	}

	// Verify that only permission bits are set.
	if (mode & filesystem.ModePermissionsMask) != mode {
		return errors.New("non-permission bits detected in directory mode")
	}

	// Success.
	return nil
}

// markExecutableForReaders sets the executable bit for the mode for any case
// where the corresponding read bit is set. It's worth noting that we implement
// this function as three separate checks (rather than a mask with a bit shift
// that's or'ed into the result) because we don't want to assume anything about
// the layout of permission bits.
func markExecutableForReaders(mode filesystem.Mode) filesystem.Mode {
	// Set the user executable bit if necessary.
	if (mode & filesystem.ModePermissionUserRead) != 0 {
		mode |= filesystem.ModePermissionUserExecute
	}

	// Set the group executable bit if necessary.
	if (mode & filesystem.ModePermissionGroupRead) != 0 {
		mode |= filesystem.ModePermissionGroupExecute
	}

	// Set the others executable bit if necessary.
	if (mode & filesystem.ModePermissionOthersRead) != 0 {
		mode |= filesystem.ModePermissionOthersExecute
	}

	// Done.
	return mode
}
