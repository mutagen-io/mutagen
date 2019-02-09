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
	// composedFileNamePrefix is the prefix used for temporary files created by
	// the Unicode decomposition test. It is in NFC form.
	composedFileNamePrefix = TemporaryNamePrefix + "unicode-test-\xc3\xa9ntry"
	// decomposedFileNamePrefix is the NFD equivalent of composedFileNamePrefix.
	decomposedFileNamePrefix = TemporaryNamePrefix + "unicode-test-\x65\xcc\x81ntry"
)

// DecomposesUnicodeByPath determines whether or not the filesystem on which the
// directory at the specified path resides decomposes Unicode filenames.
func DecomposesUnicodeByPath(path string) (bool, error) {
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
	contents, err := DirectoryContentsByPath(path)
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
func DecomposesUnicode(directory *Directory) (bool, error) {
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

	// Grab the content names in the directory.
	names, err := directory.ReadContentNames()
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
