package sync

import (
	"github.com/pkg/errors"
)

func (c *Cache) EnsureValid() error {
	// A nil cache is considered valid (though obviously that requires using
	// the GetEntries accessor).
	if c == nil {
		return errors.New("nil cache")
	}

	// Technically we could validate each path, but that's error prone,
	// expensive, and not really needed for memory safety.

	// Nil cache entries are invalid.
	for _, e := range c.Entries {
		if e == nil {
			return errors.New("nil cache entry detected")
		}
	}

	// Success.
	return nil
}
