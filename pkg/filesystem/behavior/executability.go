package behavior

import (
	"fmt"
	"os"
	"runtime"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
)

const (
	// executabilityProbeFileNamePrefix is the prefix used for temporary files
	// created by the executability preservation test.
	executabilityProbeFileNamePrefix = filesystem.TemporaryNamePrefix + "executability-test"
)

// PreservesExecutabilityByPath determines whether or not the filesystem on
// which the directory at the specified path resides preserves POSIX
// executability bits. It allows for the path leaf to be a symbolic link. The
// second value returned by this function indicates whether or not probe files
// were used in determining behavior.
func PreservesExecutabilityByPath(path string, probeMode ProbeMode, logger *logging.Logger) (bool, bool, error) {
	// Check the filesystem probing mode and see if we can return an assumption.
	if probeMode == ProbeMode_ProbeModeAssume {
		return assumeExecutabilityPreservation, false, nil
	} else if !probeMode.Supported() {
		panic("invalid probe mode")
	}

	// Check if we have a fast test that will work. If we're on Windows, we
	// enforce that the fast path was used. There is some code below, namely the
	// use of os.File's Chmod method (and possibly the os.File's Stat method,
	// which may be racey on Windows), that won't work on Windows (though it
	// could possibly be adapted in case we add a force-probe probe mode), which
	// is why we require that the fast path succeeds on Windows.
	if result, ok := probeExecutabilityPreservationFastByPath(path); ok {
		return result, false, nil
	} else if runtime.GOOS == "windows" {
		panic("fast path not used on Windows")
	}

	// Create a temporary file.
	file, err := os.CreateTemp(path, executabilityProbeFileNamePrefix)
	if err != nil {
		return false, true, fmt.Errorf("unable to create test file: %w", err)
	}

	// Ensure that the file is cleaned up and removed when we're done.
	defer func() {
		file.Close()
		must.OSRemove(file.Name(), logger)
	}()

	// Mark the file as user-executable. We use the os.File-based Chmod here
	// since this code only runs on POSIX systems where this is supported.
	if err = file.Chmod(0700); err != nil {
		return false, true, fmt.Errorf("unable to mark test file as executable: %w", err)
	}

	// Grab the file statistics and check for executability. We enforce that
	// only the user-executable bit is set, because filesystems that don't
	// preserve executability on POSIX systems (e.g. FAT32 on older versions of
	// Darwin) sometimes mark every file as having every executable bit set,
	// which is another type of non-preserving behavior. This behavior is not
	// universal (e.g. FAT32 on Linux marks every file as having no executable
	// bit set), but this test should catch many of these cases. It is, however,
	// susceptible to umask settings (e.g. umask 0022), so it's not foolproof.
	if info, err := file.Stat(); err != nil {
		return false, true, fmt.Errorf("unable to check test file executability: %w", err)
	} else {
		return info.Mode()&0111 == 0100, true, nil
	}
}

// PreservesExecutability determines whether or not the specified directory (and
// its underlying filesystem) preserves POSIX executability bits. The second
// value returned by this function indicates whether or not probe files were
// used in determining behavior.
func PreservesExecutability(directory *filesystem.Directory, probeMode ProbeMode, logger *logging.Logger) (bool, bool, error) {
	// Check the filesystem probing mode and see if we can return an assumption.
	if probeMode == ProbeMode_ProbeModeAssume {
		return assumeExecutabilityPreservation, false, nil
	} else if !probeMode.Supported() {
		panic("invalid probe mode")
	}

	// Check if we have a fast test that will work. If we're on Windows, we
	// enforce that the fast path was used. There is some code below, namely the
	// use of os.File's Chmod method (and possibly the os.File's Stat method,
	// which may be racey on Windows), that won't work on Windows (though it
	// could possibly be adapted in case we add a force-probe probe mode), which
	// is why we require that the fast path succeeds on Windows.
	if result, ok := probeExecutabilityPreservationFast(directory); ok {
		return result, false, nil
	} else if runtime.GOOS == "windows" {
		panic("fast path not used on Windows")
	}

	// Create a temporary file.
	name, file, err := directory.CreateTemporaryFile(executabilityProbeFileNamePrefix)
	if err != nil {
		return false, true, fmt.Errorf("unable to create test file: %w", err)
	}

	// Ensure that the file is cleaned up and removed when we're done.
	defer func() {
		must.Close(file, logger)
		must.RemoveFile(directory, name, logger)
	}()

	// HACK: Convert the file to an os.File object for race-free Chmod and Stat
	// access. This is an acceptable hack since we live inside the same package
	// as the Directory implementation.
	osFile, ok := file.(*os.File)
	if !ok {
		panic("opened file is not an os.File object")
	}

	// Mark the file as user-executable. We use the os.File-based Chmod here
	// since this code only runs on POSIX systems where this is supported.
	if err = osFile.Chmod(0700); err != nil {
		return false, true, fmt.Errorf("unable to mark test file as executable: %w", err)
	}

	// Grab the file statistics and check for executability. We enforce that
	// only the user-executable bit is set, because filesystems that don't
	// preserve executability on POSIX systems (e.g. FAT32 on Darwin) sometimes
	// mark every file as having every executable bit set, which is another type
	// of non-preserving behavior. This behavior is not universal (e.g. FAT32 on
	// Linux marks every file as having no executable bit set), but this test
	// should be.
	if info, err := osFile.Stat(); err != nil {
		return false, true, fmt.Errorf("unable to check test file executability: %w", err)
	} else {
		return info.Mode()&0111 == 0100, true, nil
	}
}
