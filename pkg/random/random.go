package random

import (
	"crypto/rand"

	"github.com/pkg/errors"
)

const (
	// CollisionResistantLength is the number of random bytes needed to ensure
	// collision-resistance in an identifier.
	CollisionResistantLength = 32
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
