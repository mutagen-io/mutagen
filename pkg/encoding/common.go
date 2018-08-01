package encoding

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

// loadAndUnmarshal provides the underlying loading and unmarshaling
// functionality for the encoding package.
func loadAndUnmarshal(path string, unmarshal func([]byte) error) error {
	// Grab the file contents.
	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return errors.Wrap(err, "unable to load file")
	}

	// Perform the unmarshaling.
	if err := unmarshal(data); err != nil {
		return errors.Wrap(err, "unable to unmarshal data")
	}

	// Success.
	return nil
}

// marshalAndSave provide the underlying marshaling and saving functionality for
// the encoding package.
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
