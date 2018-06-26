package sync

import (
	"os"
)

const (
	// newDirectoryBaseMode is the base mode for directories created in
	// transitions.
	newDirectoryBaseMode os.FileMode = 0700

	// newFileBaseMode is the base mode for files created in transitions.
	newFileBaseMode os.FileMode = 0600
)

// anyExecutableBitSet returns true if any executable bit is set on the file,
// false otherwise.
func anyExecutableBitSet(mode os.FileMode) bool {
	return mode&0111 != 0
}

// stripExecutableBits strips all executability bits from the specified file
// mode.
func stripExecutableBits(mode os.FileMode) os.FileMode {
	return mode &^ 0111
}

// markExecutableForReaders sets the executable bit for the mode for any case
// where the corresponding read bit is set.
func markExecutableForReaders(mode os.FileMode) os.FileMode {
	// Set the user executable bit if necessary.
	if mode&0400 != 0 {
		mode |= 0100
	}

	// Set the group executable bit if necessary.
	if mode&0040 != 0 {
		mode |= 0010
	}

	// Set the other executable bit if necessary.
	if mode&0004 != 0 {
		mode |= 0001
	}

	// Done.
	return mode
}
