package tunneling

import (
	"errors"
)

// EnsureValid ensures that Configuration's invariants are respected.
func (c *Configuration) EnsureValid() error {
	// Ensure that the configuration is non-nil.
	if c == nil {
		return errors.New("nil configuration")
	}

	// Success.
	return nil
}
