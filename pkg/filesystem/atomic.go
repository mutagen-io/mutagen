package filesystem

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	// atomicWriteTemporaryNamePrefix is the file name prefix to use for
	// intermediate temporary files used in atomic writes.
	atomicWriteTemporaryNamePrefix = TemporaryNamePrefix + "atomic-write"
)

// WriteFileAtomic writes a file to disk in an atomic fashion by using an
// intermediate temporary file that is swapped in place using a rename
// operation.
func WriteFileAtomic(path string, data []byte, permissions os.FileMode) error {
	// Create a temporary file. The ioutil module already uses secure
	// permissions for creating the temporary file, so we don't need to specify
	// any.
	temporary, err := ioutil.TempFile(filepath.Dir(path), atomicWriteTemporaryNamePrefix)
	if err != nil {
		return errors.Wrap(err, "unable to create temporary file")
	}

	// Write data.
	if _, err = temporary.Write(data); err != nil {
		temporary.Close()
		os.Remove(temporary.Name())
		return errors.Wrap(err, "unable to write data to temporary file")
	}

	// Close out the file.
	if err = temporary.Close(); err != nil {
		os.Remove(temporary.Name())
		return errors.Wrap(err, "unable to close temporary file")
	}

	// Set the file's permissions.
	if err = os.Chmod(temporary.Name(), permissions); err != nil {
		os.Remove(temporary.Name())
		return errors.Wrap(err, "unable to change file permissions")
	}

	// Rename the file.
	if err = os.Rename(temporary.Name(), path); err != nil {
		os.Remove(temporary.Name())
		return errors.Wrap(err, "unable to rename file")
	}

	// Success.
	return nil
}
