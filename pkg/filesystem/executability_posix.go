// +build !windows

package filesystem

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/errors"
)

const (
	// executabilityTestFilenamePrefix is the prefix used for temporary files
	// created by the executability preservation test.
	executabilityTestFilenamePrefix = ".mutagen-executability-test"
)

// IsExecutabilityProbeFileName determines whether or not a file name (not a
// file path) is the name of an executability preservation probe file.
func IsExecutabilityProbeFileName(name string) bool {
	return strings.HasPrefix(name, executabilityTestFilenamePrefix)
}

// PreservesExecutability determines whether or not the filesystem on which the
// directory at the specified path resides preserves POSIX executability bits.
func PreservesExecutability(path string) (bool, error) {
	// Create a temporary file.
	file, err := ioutil.TempFile(path, executabilityTestFilenamePrefix)
	if err != nil {
		return false, errors.Wrap(err, "unable to create test file")
	}

	// Ensure that the file is cleaned up and removed when we're done.
	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()

	// Mark the file as user executable. We use the os.File-based Chmod here
	// since this code only runs on POSIX systems where this is supported.
	if err = file.Chmod(0700); err != nil {
		return false, errors.Wrap(err, "unable to mark test file as executable")
	}

	// Grab the file statistics and check for executability. We enforce that
	// only the user-executable bit is set, because filesystems that don't
	// preserve executability on POSIX systems (e.g. FAT32 on Darwin) sometimes
	// mark every file as having every executable bit set, which is another type
	// of non-preserving behavior. This behavior is not universal (e.g. FAT32 on
	// Linux marks every file as having no executable bit set), but this test
	// should be.
	if info, err := file.Stat(); err != nil {
		return false, errors.Wrap(err, "unable to check test file executability")
	} else {
		return info.Mode()&0111 == 0100, nil
	}
}
