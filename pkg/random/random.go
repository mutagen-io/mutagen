package random

import (
	"crypto/rand"
	"fmt"
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
		return nil, fmt.Errorf("unable to read random data: %w", err)
	}

	// Success.
	return result, nil
}
