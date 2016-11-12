package sync

import (
	"bytes"
)

// equalShallow returns true if and only if the existence, kind, executability,
// and digest of the two entries are equivalent. It pays no attention to the
// contents of either entry.
func (e *Entry) equalShallow(other *Entry) bool {
	// If both are nil, they can be considered equal.
	if e == nil && other == nil {
		return true
	}

	// If only one is nil, they can't be equal.
	if e == nil || other == nil {
		return false
	}

	// Check properties.
	return e.Kind == other.Kind &&
		e.Executable == other.Executable &&
		bytes.Equal(e.Digest, other.Digest)
}

func (e *Entry) Equal(other *Entry) bool {
	// Verify that the entries are shallow equal first.
	if !e.equalShallow(other) {
		return false
	}

	// Compare contents.
	if len(e.Contents) != len(other.Contents) {
		return false
	}
	for n, c := range e.Contents {
		if !c.Equal(other.Contents[n]) {
			return false
		}
	}

	// Success.
	return true
}

func (e *Entry) copyShallow() *Entry {
	// If the entry is nil, the copy is nil.
	if e == nil {
		return nil
	}

	// Create the copy.
	return &Entry{
		Kind:       e.Kind,
		Executable: e.Executable,
		Digest:     e.Digest,
	}
}

func (e *Entry) copy() *Entry {
	// If the entry is nil, the copy is nil.
	if e == nil {
		return nil
	}

	// Create the result.
	result := &Entry{
		Kind:       e.Kind,
		Executable: e.Executable,
		Digest:     e.Digest,
	}

	// Copy contents, if any.
	if len(e.Contents) > 0 {
		result.Contents = make(map[string]*Entry, len(e.Contents))
		for n, c := range e.Contents {
			result.Contents[n] = c
		}
	}

	// Done.
	return result
}
