package core

import (
	"bytes"
	"errors"
	"fmt"
)

// EnsureValid ensures that Cache's invariants are respected.
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
			return errors.New("cache entry with nil modification time detected")
		} else if err := e.ModificationTime.CheckValid(); err != nil {
			return fmt.Errorf("cache entry modification time invalid: %w", err)
		}
	}

	// Success.
	return nil
}

// Equal determines whether or not another cache is equal to this one. It is
// designed specifically for tests, though it is exported so that it can be used
// by scan_bench.
func (c *Cache) Equal(other *Cache) bool {
	// Verify non-nilness. We don't consider nil caches valid, so we don't
	// consider them equal.
	if c == nil || other == nil {
		return false
	}

	// Handle equivalence fast paths.
	if c == other {
		return true
	}

	// Check lengths.
	if len(c.Entries) != len(other.Entries) {
		return false
	}

	// Check contents.
	for path, entry := range c.Entries {
		// Extract corresponding content.
		otherEntry, ok := other.Entries[path]
		if !ok {
			return false
		}

		// Watch for nil values as a sanity check.
		if entry == nil || otherEntry == nil {
			panic("cache has nil entry")
		} else if entry.ModificationTime == nil || otherEntry.ModificationTime == nil {
			panic("nil modification time in cache")
		}

		// Verify equivalence
		equivalent := otherEntry.Mode == entry.Mode &&
			otherEntry.ModificationTime.Seconds == entry.ModificationTime.Seconds &&
			otherEntry.ModificationTime.Nanos == entry.ModificationTime.Nanos &&
			otherEntry.Size == entry.Size &&
			otherEntry.FileID == entry.FileID &&
			bytes.Equal(otherEntry.Digest, entry.Digest)
		if !equivalent {
			return false
		}
	}

	// Success.
	return true
}

// byteLookupMap is the interface implemented by all byteLookupMap types.
type byteLookupMap interface {
	// length returns the length of the map.
	length() int
	// insert adds a key-value pair to the map.
	insert(k []byte, v string)
	// find looks for a key in the map, returning the associated value
	// (defaulting to an empty string if the key was not present) and whether or
	// not the key was found.
	find(k []byte) (string, bool)
}

// ReverseLookupMap provides facilities for doing reverse lookups to avoid
// expensive staging operations in the case of renames and copies.
type ReverseLookupMap struct {
	// lookupMap is the underlying map.
	lookupMap byteLookupMap
}

// Length returns the number of entries in the map.
func (m *ReverseLookupMap) Length() int {
	return m.lookupMap.length()
}

// Lookup attempts a lookup in the map.
func (m *ReverseLookupMap) Lookup(digest []byte) (string, bool) {
	return m.lookupMap.find(digest)
}

// GenerateReverseLookupMap creates a reverse lookup map from a cache.
func (c *Cache) GenerateReverseLookupMap() (*ReverseLookupMap, error) {
	// Create a placeholder for the map that we're going to initialize.
	var lookupMap byteLookupMap

	// Track the digest size and ensure it's consistent.
	digestSize := -1

	// Loop over entries.
	for p, e := range c.Entries {
		// Compute and validate the digest size and allocate the map.
		if digestSize == -1 {
			digestSize = len(e.Digest)
			if digestSize == 20 {
				lookupMap = make(byteLookupMap20, len(c.Entries))
			} else if digestSize == 32 {
				lookupMap = make(byteLookupMap32, len(c.Entries))
			} else if digestSize == 16 {
				lookupMap = make(byteLookupMap16, len(c.Entries))
			} else {
				return nil, errors.New("unsupported digest size")
			}
		} else if len(e.Digest) != digestSize {
			return nil, errors.New("inconsistent digest sizes")
		}

		// Insert the entry.
		lookupMap.insert(e.Digest, p)
	}

	// If there are no entries, then we'll still need a lookup map.
	if len(c.Entries) == 0 {
		lookupMap = &emptyByteLookupMap{}
	}

	// Success.
	return &ReverseLookupMap{lookupMap}, nil
}
