package behavior

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

const (
	// composedFileNamePrefix is the prefix used for temporary files created by
	// the Unicode decomposition test. It is in NFC form.
	composedFileNamePrefix = filesystem.TemporaryNamePrefix + "unicode-test-\xc3\xa9ntry"
	// decomposedFileNamePrefix is the NFD equivalent of composedFileNamePrefix.
	decomposedFileNamePrefix = filesystem.TemporaryNamePrefix + "unicode-test-\x65\xcc\x81ntry"
)

// DecomposesUnicodeByPath determines whether or not the filesystem on which the
// directory at the specified path resides decomposes Unicode filenames.
func DecomposesUnicodeByPath(path string, probeMode ProbeMode) (bool, error) {
	// Check the filesystem probing mode and see if we can return an assumption.
	if probeMode == ProbeMode_ProbeModeAssume {
		return assumeUnicodeDecomposition, nil
	} else if !probeMode.Supported() {
		panic("invalid probe mode")
	}

	// Check if we have a fast test that will work.
	if result, ok := probeUnicodeDecompositionFastByPath(path); ok {
		return result, nil
	}

	// Create and close a temporary file using the composed filename.
	file, err := ioutil.TempFile(path, composedFileNamePrefix)
	if err != nil {
		return false, errors.Wrap(err, "unable to create test file")
	} else if err = file.Close(); err != nil {
		return false, errors.Wrap(err, "unable to close test file")
	}

	// Grab the file's name. This is calculated from the parameters passed to
	// TempFile, not by reading from the OS, so it will still be in a composed
	// form. Also calculate a decomposed variant.
	composedFilename := filepath.Base(file.Name())
	decomposedFilename := strings.Replace(
		composedFilename,
		composedFileNamePrefix,
		decomposedFileNamePrefix,
		1,
	)

	// Defer removal of the file. Since we don't know whether the filesystem is
	// also normalization-insensitive, we try both compositions.
	defer func() {
		if os.Remove(filepath.Join(path, composedFilename)) != nil {
			os.Remove(filepath.Join(path, decomposedFilename))
		}
	}()

	// Grab the contents of the path.
	contents, err := filesystem.DirectoryContentsByPath(path)
	if err != nil {
		return false, errors.Wrap(err, "unable to read directory contents")
	}

	// Loop through contents and see if we find a match for the decomposed file
	// name. It doesn't even need to be our file, though it probably will be.
	for _, c := range contents {
		name := c.Name()
		if name == decomposedFilename {
			return true, nil
		} else if name == composedFilename {
			return false, nil
		}
	}

	// If we didn't find any match, something's fishy.
	return false, errors.New("unable to find test file after creation")
}

// DecomposesUnicode determines whether or not the specified directory (and its
// underlying filesystem) decomposes Unicode filenames.
func DecomposesUnicode(directory *filesystem.Directory, probeMode ProbeMode) (bool, error) {
	// Check the filesystem probing mode and see if we can return an assumption.
	if probeMode == ProbeMode_ProbeModeAssume {
		return assumeUnicodeDecomposition, nil
	} else if !probeMode.Supported() {
		panic("invalid probe mode")
	}

	// Check if we have a fast test that will work.
	if result, ok := probeUnicodeDecompositionFast(directory); ok {
		return result, nil
	}

	// Create and close a temporary file using the composed filename.
	composedName, file, err := directory.CreateTemporaryFile(composedFileNamePrefix)
	if err != nil {
		return false, errors.Wrap(err, "unable to create test file")
	} else if err = file.Close(); err != nil {
		return false, errors.Wrap(err, "unable to close test file")
	}

	// The name returned from CreateTemporaryFile is calculated from the
	// provided pattern, so it will still be in a composed form. Compute the
	// decomposed variant.
	decomposedName := strings.Replace(
		composedName,
		composedFileNamePrefix,
		decomposedFileNamePrefix,
		1,
	)

	// Defer removal of the file. Since we don't know whether the filesystem is
	// also normalization-insensitive, we try both compositions.
	defer func() {
		if directory.RemoveFile(composedName) != nil {
			directory.RemoveFile(decomposedName)
		}
	}()

	// HACK: If we're on Linux, then re-open the directory after creating the
	// temporary file (and defer closure of the re-opened copy). This is
	// necessary to work around an issue with osxfs where a directory descriptor
	// can't be used to list contents created after the descriptor was opened
	// (due either to aggressive caching or some sort of implementation bug).
	// See issue #73 for more details. Ideally we'd restrict this workaround to
	// osxfs, but we can't actually detect osxfs specifically because the statfs
	// type field just indicates that it's a FUSE filesystem. Even if we wanted
	// to restrict this behavior to just FUSE filesystems, the statfs call is
	// going to be about the same cost (if not more expensive) than the re-open
	// call, so it's best to just do this in all cases on Linux. This isn't such
	// a big deal since this function is only called once per scan, and we may
	// hit a fast path above anyway.
	directoryForContentRead := directory
	if runtime.GOOS == "linux" {
		directoryForContentRead, err = directory.OpenDirectory(".")
		if err != nil {
			return false, errors.Wrap(err, "unable to re-open directory")
		}
		defer directoryForContentRead.Close()
	}

	// Grab the content names in the directory.
	names, err := directoryForContentRead.ReadContentNames()
	if err != nil {
		return false, errors.Wrap(err, "unable to read directory content names")
	}

	// Loop through the names and see if we find a match for either the composed
	// or decomposed name.
	for _, name := range names {
		if name == decomposedName {
			return true, nil
		} else if name == composedName {
			return false, nil
		}
	}

	// If we didn't find any match, something's fishy.
	return false, errors.New("unable to find test file after creation")
}
