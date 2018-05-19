package sync

import (
	"bytes"

	"github.com/pkg/errors"
)

func (e *Entry) EnsureValid() error {
	// If the entry is nil, it's technically valid, though only for roots.
	if e == nil {
		return nil
	}

	// Otherwise validate based on kind.
	if e.Kind == EntryKind_Directory {
		// Ensure that no invalid fields are set.
		if e.Executable {
			return errors.New("executable directory detected")
		} else if e.Digest != nil {
			return errors.New("non-nil directory digest detected")
		} else if e.Target != "" {
			return errors.New("non-empty symlink target detected for directory")
		}

		// Validate contents. Nil entries are NOT allowed as contents.
		for name, entry := range e.Contents {
			if name == "" {
				return errors.New("empty content name detected")
			} else if entry == nil {
				return errors.New("nil content detected")
			} else if err := entry.EnsureValid(); err != nil {
				return err
			}
		}
	} else if e.Kind == EntryKind_File {
		// Ensure that no invalid fields are set.
		if e.Contents != nil {
			return errors.New("non-nil file contents detected")
		} else if e.Target != "" {
			return errors.New("non-empty symlink target detected for file")
		}

		// Ensure that the digest is non-empty.
		if len(e.Digest) == 0 {
			return errors.New("file with empty digest detected")
		}
	} else if e.Kind == EntryKind_Symlink {
		// Ensure that no invalid fields are set.
		if e.Executable {
			return errors.New("executable symlink detected")
		} else if e.Digest != nil {
			return errors.New("non-nil symlink digest detected")
		} else if e.Contents != nil {
			return errors.New("non-nil symlink contents detected")
		}

		// Ensure that the target is non-empty.
		if e.Target == "" {
			return errors.New("symlink with empty target detected")
		}
	} else {
		return errors.New("unknown entry kind detected")
	}

	// Success.
	return nil
}

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
		bytes.Equal(e.Digest, other.Digest) &&
		e.Target == other.Target
}

func (e *Entry) Equal(other *Entry) bool {
	// Verify that the entries are shallow equal first.
	if !e.equalShallow(other) {
		return false
	}

	// If both are nil, then we're done.
	if e == nil && other == nil {
		return true
	}

	// Compare contents.
	if len(e.Contents) != len(other.Contents) {
		return false
	}
	for name, entry := range e.Contents {
		otherEntry, ok := other.Contents[name]
		if !ok || !entry.Equal(otherEntry) {
			return false
		}
	}

	// Success.
	return true
}

func (e *Entry) CopyShallow() *Entry {
	// If the entry is nil, the copy is nil.
	if e == nil {
		return nil
	}

	// Create the shallow copy.
	return &Entry{
		Kind:       e.Kind,
		Executable: e.Executable,
		Digest:     e.Digest,
		Target:     e.Target,
	}
}

func (e *Entry) Copy() *Entry {
	// If the entry is nil, the copy is nil.
	if e == nil {
		return nil
	}

	// Create the result.
	result := &Entry{
		Kind:       e.Kind,
		Executable: e.Executable,
		Digest:     e.Digest,
		Target:     e.Target,
	}

	// If the original entry doesn't have any contents, return now to save an
	// allocation.
	if len(e.Contents) == 0 {
		return result
	}

	// Copy contents.
	result.Contents = make(map[string]*Entry)
	for name, entry := range e.Contents {
		result.Contents[name] = entry.Copy()
	}

	// Done.
	return result
}
