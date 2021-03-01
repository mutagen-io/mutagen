package encoding

import (
	"os"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

// LoadAndUnmarshal provides the underlying loading and unmarshaling
// functionality for the encoding package. It reads the data at the specified
// path and then invokes the specified unmarshaling callback (usually a closure)
// to decode the data.
func LoadAndUnmarshal(path string, unmarshal func([]byte) error) error {
	// Grab the file contents.
	data, err := os.ReadFile(path)
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

// MarshalAndSave provide the underlying marshaling and saving functionality for
// the encoding package. It invokes the specified marshaling callback (usually a
// closure) and writes the result atomically to the specified path. The data is
// saved with read/write permissions for the user only.
func MarshalAndSave(path string, marshal func() ([]byte, error)) error {
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
