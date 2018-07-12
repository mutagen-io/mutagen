// +build !windows

package filesystem

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

const (
	decompositionTestFilenamePrefix = ".mutagen-decomposition-test-"
	composedFilenamePrefix          = ".mutagen-decomposition-test-\xc3\xa9ntry"
	decomposedFilenamePrefix        = ".mutagen-decomposition-test-\x65\xcc\x81ntry"
)

func isDecompositionTestPath(path string) bool {
	return strings.HasPrefix(filepath.Base(path), decompositionTestFilenamePrefix)
}

func DecomposesUnicode(path string) (bool, error) {
	// Create and close a temporary file using the composed filename.
	file, err := ioutil.TempFile(path, composedFilenamePrefix)
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
		composedFilenamePrefix,
		decomposedFilenamePrefix,
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
	contents, err := DirectoryContents(path)
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
