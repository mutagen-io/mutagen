package filesystem

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func WriteFileAtomic(path string, data []byte, permissions os.FileMode) error {
	// Create a temporary file. The ioutil module already uses secure
	// permissions for creating the temporary file, so we don't need to specify
	// any.
	dirname, basename := filepath.Split(path)
	temporary, err := ioutil.TempFile(dirname, basename)
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
