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
	// expensive, and not really needed for memory safety. Also note that an
	// empty path is valid when the synchronization root is a file.

	// Nil cache entries are invalid.
	for _, e := range c.Entries {
		if e == nil {
			return errors.New("nil cache entry detected")
		} else if e.ModificationTime == nil {
			return errors.New("cache entry will nil modification time detected")
		}
	}

	// Success.
	return nil
}

type ReverseLookupMap struct {
	// map20 provides mappings for SHA-1 hashes.
	map20 map[[20]byte]string
}

func (m *ReverseLookupMap) Lookup(digest []byte) (string, bool) {
	// Handle based on digest length.
	if len(digest) == 20 {
		// Create a key.
		var key [20]byte
		copy(key[:], digest)

		// Attempt a lookup.
		result, ok := m.map20[key]

		// Done.
		return result, ok
	}

	// If the digest wasn't of a supported length, then there's no harm.
	return "", false
}

func (c *Cache) GenerateReverseLookupMap() (*ReverseLookupMap, error) {
	// Create the map.
	result := &ReverseLookupMap{}

	// Track the digest size and ensure it's consistent.
	digestSize := -1

	// Loop over entries.
	for p, e := range c.Entries {
		// Compute and validate the digest size and allocate the map.
		if digestSize == -1 {
			digestSize = len(e.Digest)
			if digestSize == 20 {
				result.map20 = make(map[[20]byte]string, len(c.Entries))
			} else {
				return nil, errors.New("unsupported digest size")
			}
		} else if len(e.Digest) != digestSize {
			return nil, errors.New("inconsistent digest sizes")
		}

		// Create a key.
		var key [20]byte
		copy(key[:], e.Digest)

		// Store the entry.
		if digestSize == 20 {
			result.map20[key] = p
		} else {
			panic("invalid digest size allowed")
		}
	}

	// Success.
	return result, nil
}
