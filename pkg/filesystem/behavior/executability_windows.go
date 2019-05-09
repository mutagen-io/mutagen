package behavior

import (
	"github.com/havoc-io/mutagen/pkg/filesystem"
)

// PreservesExecutabilityByPath determines whether or not the directory at the
// specified path preserves POSIX executability bits. On Windows this function
// always returns false since POSIX executability bits are never preserved.
func PreservesExecutabilityByPath(_ string, probeMode ProbeMode) (bool, error) {
	// Check for invalid probe modes.
	if !probeMode.Supported() {
		panic("invalid probe mode")
	}

	// Return the well-known behavior.
	return false, nil
}

// PreservesExecutability determines whether or not the specified directory (and
// its underlying filesystem) preserves POSIX executability bits. On Windows
// this function always returns false since POSIX executability bits are never
// preserved.
func PreservesExecutability(_ *filesystem.Directory, probeMode ProbeMode) (bool, error) {
	// Check for invalid probe modes.
	if !probeMode.Supported() {
		panic("invalid probe mode")
	}

	// Return the well-known behavior.
	return false, nil
}
