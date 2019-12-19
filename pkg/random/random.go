package random

import (
	"crypto/rand"
	"fmt"
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
