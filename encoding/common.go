package encoding

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/filesystem"
)

func loadAndUnmarshal(path string, unmarshal func([]byte) error) error {
	// Grab the file contents.
	data, err := ioutil.ReadFile(path)
	if err != nil {
		// If this is a non-existence error, then return it without wrapping so
		// that the caller and check for it.
		if os.IsNotExist(err) {
			return err
		}

		// Otherwise wrap it up.
		return errors.Wrap(err, "unable to load file")
	}

	// Perform the unmarshaling.
	if err := unmarshal(data); err != nil {
		return errors.Wrap(err, "unable to unmarshal data")
	}

	// Success.
	return nil
}

func marshalAndSave(path string, marshal func() ([]byte, error)) error {
	// Marshal the message.
	data, err := marshal()
	if err != nil {
		return errors.Wrap(err, "unable to marshal message")
	}

	// Write the file atomically with secure file permissions.
	if err := filesystem.WriteFileAtomic(path, data, 0600); err != nil {
		return errors.Wrap(err, "unable to write message data")
	}

	// Success.
	return nil
}
