package sync

import (
	"bytes"
	"strings"

	"github.com/pkg/errors"
)

// EnsureValid ensures that Entry's invariants are respected.
func (e *Entry) EnsureValid() error {
	// A nil entry is technically valid, at least in certain contexts. It
	// represents the absence of content.
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
			} else if name == "." || name == ".." {
				return errors.New("dot name detected")
			} else if strings.IndexByte(name, '/') != -1 {
				return errors.New("content name contains path separator")
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

		// We intentionally avoid any validation on the symlink target itself
		// because there's no validation that we can perform in POSIX raw mode.
	} else {
		return errors.New("unknown entry kind detected")
	}

	// Success.
	return nil
}

// entryVisitor is a callback type used for Entry.walk. It receives two
// arguments: the path of the entry within the entry hierarchy and the entry
// itself.
type entryVisitor func(string, *Entry)

// walk performs a DFS-traversal of the entry, invoking the specified visitor on
// each element. The path argument specifies the path at which the entry should
// be treated as residing.
func (e *Entry) walk(path string, visitor entryVisitor) {
	// Otherwise visit ourselves.
	visitor(path, e)

	// If we're non-nil and a directory, visit our children.
	if e != nil && e.Kind == EntryKind_Directory {
		for name, entry := range e.Contents {
			entry.walk(pathJoin(path, name), visitor)
		}
	}
}

// Count returns the total number of entries within the entry hierarchy rooted
// at the entry.
func (e *Entry) Count() uint64 {
	// If we're a nil entry, then the hierarchy is empty.
	if e == nil {
		return 0
	}

	// Count ourselves.
	result := uint64(1)

	// If we're a directory, count our children.
	if e.Kind == EntryKind_Directory {
		for _, entry := range e.Contents {
			// TODO: At the moment, we don't worry about overflow here. The
			// reason is that, in order to overflow uint64, we'd need a minimum
			// of 2**64 entries in the hierarchy. Even assuming that each entry
			// consumed only one byte of memory (and they consume at least an
			// order of magnitude more than that), we'd have to be on a system
			// with (at least) ~18.5 exabytes of memory. Additionally, Protocol
			// Buffers messages have even lower size limits that would prevent
			// such an Entry from being sent over the network. But we should
			// still fix this at some point.
			result += entry.Count()
		}
	}

	// Done.
	return result
}

// equalShallow returns true if and only if the existence, kind, executability,
// and digest of the two entries are equivalent. It pays no attention to the
// contents of either entry.
func (e *Entry) equalShallow(other *Entry) bool {
	// If the pointers are equal, then the entries are equal. Even in the case
	// of two nil pointers, we still consider the entries to be equal since they
	// both express absence.
	if e == other {
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

// Equal determines whether or not another entry is entirely (recursively) equal
// to this one.
func (e *Entry) Equal(other *Entry) bool {
	// If the pointers are equal, then the entries are equal. Even in the case
	// of two nil pointers, we still consider the entries to be equal since they
	// both express absence.
	if e == other {
		return true
	}

	// Verify that the entries are shallow equal first.
	if !e.equalShallow(other) {
		return false
	}

	// At this point, we know that both pointers are non-nil, because shallow
	// equivalence ensures that either both pointers are nil or both pointers
	// are non-nil, and we exclude the both-nil case above.

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

// copySlim creates a "slim" copy of the entry. For files and symbolic links,
// this yields an equivalent entry. For directories, it yields an equivalent
// entry but without any contents.
func (e *Entry) copySlim() *Entry {
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

// Copy creates a deep copy of the entry.
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
