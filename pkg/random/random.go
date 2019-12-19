package random

import (
	"crypto/rand"

	"github.com/pkg/errors"
)

// New returns a byte slice of the specified length with cryptographically
// random conents.
func New(length int) ([]byte, error) {
	// Create the buffer.
	result := make([]byte, length)

	// Read random data.
	if _, err := rand.Read(result[:]); err != nil {
		return nil, errors.Wrap(err, "unable to read random data")
	}

	// Success.
	return result, nil
}
