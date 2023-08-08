package core

import (
	"bytes"
	"errors"
	"strings"

	"github.com/mutagen-io/mutagen/pkg/synchronization/core/fastpath"
)

// synchronizable returns true if the entry kind is synchronizable and false if
// the entry kind is unsynchronizable.
func (k EntryKind) synchronizable() bool {
	return k == EntryKind_Directory ||
		k == EntryKind_File ||
		k == EntryKind_SymbolicLink
}

// MarshalText implements encoding.TextMarshaler.MarshalText.
func (k EntryKind) MarshalText() ([]byte, error) {
	var result string
	switch k {
	case EntryKind_Directory:
		result = "directory"
	case EntryKind_File:
		result = "file"
	case EntryKind_SymbolicLink:
		result = "symlink"
	case EntryKind_Untracked:
		result = "untracked"
	case EntryKind_Problematic:
		result = "problematic"
	case EntryKind_PhantomDirectory:
		result = "phantom-directory"
	default:
		result = "unknown"
	}
	return []byte(result), nil
}

// EnsureValid ensures that Entry's invariants are respected. If synchronizable
// is true, then unsynchronizable content will be considered invalid.
func (e *Entry) EnsureValid(synchronizable bool) error {
	// A nil entry represents an absence of content and is therefore valid.
	if e == nil {
		return nil
	}

	// Otherwise validate based on kind.
	if e.Kind == EntryKind_Directory {
		// Ensure that no invalid fields are set.
		if e.Digest != nil {
			return errors.New("non-nil directory digest detected")
		} else if e.Executable {
			return errors.New("executable directory detected")
		} else if e.Target != "" {
			return errors.New("non-empty symbolic link target detected for directory")
		} else if e.Problem != "" {
			return errors.New("non-empty problem detected for directory")
		}

		// Validate contents. Nil entries are not considered valid for contents.
		for name, entry := range e.Contents {
			if name == "" {
				return errors.New("empty content name detected")
			} else if name == "." || name == ".." {
				return errors.New("dot name detected")
			} else if strings.IndexByte(name, '/') != -1 {
				return errors.New("content name contains path separator")
			} else if entry == nil {
				return errors.New("nil content detected")
			} else if err := entry.EnsureValid(synchronizable); err != nil {
				return err
			}
		}
	} else if e.Kind == EntryKind_File {
		// Ensure that no invalid fields are set.
		if e.Contents != nil {
			return errors.New("non-nil file content map detected")
		} else if e.Target != "" {
			return errors.New("non-empty symbolic link target detected for file")
		} else if e.Problem != "" {
			return errors.New("non-empty problem detected for file")
		}

		// Ensure that the digest is non-empty.
		if len(e.Digest) == 0 {
			return errors.New("file with empty digest detected")
		}
	} else if e.Kind == EntryKind_SymbolicLink {
		// Ensure that no invalid fields are set.
		if e.Contents != nil {
			return errors.New("non-nil symbolic link content map detected")
		} else if e.Digest != nil {
			return errors.New("non-nil symbolic link digest detected")
		} else if e.Executable {
			return errors.New("executable symbolic link detected")
		} else if e.Problem != "" {
			return errors.New("non-empty problem detected for symbolic link")
		}

		// Ensure that the target is non-empty. We avoid any further validation
		// because there's none that we can reasonably perform.
		if e.Target == "" {
			return errors.New("symbolic link with empty target detected")
		}
	} else if e.Kind == EntryKind_Untracked {
		// Verify that unsynchronizable content is allowed.
		if synchronizable {
			return errors.New("untracked content is not synchronizable")
		}

		// Ensure that no invalid fields are set.
		if e.Contents != nil {
			return errors.New("non-nil untracked content map detected")
		} else if e.Digest != nil {
			return errors.New("non-nil untracked content digest detected")
		} else if e.Executable {
			return errors.New("executable untracked content detected")
		} else if e.Target != "" {
			return errors.New("non-empty symbolic link target detected for untracked content")
		} else if e.Problem != "" {
			return errors.New("non-empty problem detected for untracked content")
		}
	} else if e.Kind == EntryKind_Problematic {
		// Verify that unsynchronizable content is allowed.
		if synchronizable {
			return errors.New("problematic content is not synchronizable")
		}

		// Ensure that no invalid fields are set.
		if e.Contents != nil {
			return errors.New("non-nil problematic content map detected")
		} else if e.Digest != nil {
			return errors.New("non-nil problematic content digest detected")
		} else if e.Executable {
			return errors.New("executable problematic content detected")
		} else if e.Target != "" {
			return errors.New("non-empty symbolic link target detected for problematic content")
		}

		// Ensure that the problem is non-empty.
		if e.Problem == "" {
			return errors.New("empty problem detected for problematic content")
		}
	} else if e.Kind == EntryKind_PhantomDirectory {
		// Verify that unsynchronizable content is allowed.
		if synchronizable {
			return errors.New("phantom directory is not fully synchronizable")
		}

		// Ensure that no invalid fields are set.
		if e.Digest != nil {
			return errors.New("non-nil phantom directory digest detected")
		} else if e.Executable {
			return errors.New("executable phantom directory detected")
		} else if e.Target != "" {
			return errors.New("non-empty symbolic link target detected for phantom directory")
		} else if e.Problem != "" {
			return errors.New("non-empty problem detected for phantom directory")
		}

		// Validate contents. Nil entries are not considered valid for contents.
		for name, entry := range e.Contents {
			if name == "" {
				return errors.New("empty content name detected")
			} else if name == "." || name == ".." {
				return errors.New("dot name detected")
			} else if strings.IndexByte(name, '/') != -1 {
				return errors.New("content name contains path separator")
			} else if entry == nil {
				return errors.New("nil content detected")
			} else if err := entry.EnsureValid(synchronizable); err != nil {
				return err
			}
		}
	} else {
		return errors.New("unknown entry kind detected")
	}

	// Success.
	return nil
}

// entryVisitor is a callback type used for Entry.walk.
type entryVisitor func(path string, entry *Entry)

// walk performs a depth-first traversal of the entry, invoking the specified
// visitor on each element in the entry hierarchy. The path argument specifies
// the path at which the root entry should be treated as residing. If reverse is
// false, then each entry will be visited before its contents (i.e. a normal
// depth-first traversal), otherwise it will be visited after its contents (i.e.
// a reverse depth-first traversal).
func (e *Entry) walk(path string, visitor entryVisitor, reverse bool) {
	// If this is a normal walk, then visit the entry before its contents.
	if !reverse {
		visitor(path, e)
	}

	// If this entry is non-nil, then visit any child entries. We don't bother
	// checking if the entry is a directory since this is an internal method and
	// the caller is responsible for enforcing entry invariants (meaning that
	// only directories will have child entries).
	if e != nil {
		// Compute the prefix to add to content names to compute their paths.
		var contentPathPrefix string
		if len(e.Contents) > 0 {
			contentPathPrefix = fastpath.Joinable(path)
		}

		// Process the child entries.
		for name, child := range e.Contents {
			child.walk(contentPathPrefix+name, visitor, reverse)
		}
	}

	// If this is a reverse walk, then visit the entry after its contents.
	if reverse {
		visitor(path, e)
	}
}

// Count returns the total number of entries within the entry hierarchy rooted
// at the entry, excluding nil and unsynchronizable entries.
func (e *Entry) Count() uint64 {
	// Nil entries represent an empty hierarchy.
	if e == nil {
		return 0
	}

	// Unsynchronizable entries can be excluded from the count because they
	// don't represent content that can or will be synchronized.
	if !e.Kind.synchronizable() {
		return 0
	}

	// Count ourselves.
	result := uint64(1)

	// Count any child entries. We don't bother checking if the entry is a
	// directory since the caller is responsible for enforcing entry invariants
	// (meaning that only directories will have child entries).
	for _, child := range e.Contents {
		// TODO: At the moment, we don't worry about overflow here. The
		// reason is that, in order to overflow uint64, we'd need a minimum
		// of 2**64 entries in the hierarchy. Even assuming that each entry
		// consumed only one byte of memory (and they consume at least an
		// order of magnitude more than that), we'd have to be on a system
		// with (at least) ~18.5 exabytes of memory. Additionally, Protocol
		// Buffers messages have even lower size limits that would prevent
		// such an Entry from being sent over the network. But we should
		// still fix this at some point.
		result += child.Count()
	}

	// Done.
	return result
}

// entryEqualWildcardProblemMatch controls whether or not wildcard problem
// matching is enabled for Entry.Equal. Ideally this would be a constant so that
// the compiler could optimize away the unused branch in Entry.Equal, but
// there's no "test" build tag that we can use to redefine constants for tests
// only. The Go developers seem adamant that no such flag should be added. We
// could define one manually, but modern CPUs will chew through this additional
// check quickly enough anyway, so it's not worth the trouble.
var entryEqualWildcardProblemMatch bool

// Equal performs an equivalence comparison between this entry and another. If
// deep is true, then the comparison is performed recursively, otherwise the
// comparison is only performed between entry properties at the top level and
// content maps are ignored.
func (e *Entry) Equal(other *Entry, deep bool) bool {
	// If the pointers are equal, then the entries are equal, both shallowly and
	// recursively. This includes the case where both pointers are nil, which
	// represents the absence of content. If only one pointer is nil, then they
	// can't possibly be equal.
	if e == other {
		return true
	} else if e == nil || other == nil {
		return false
	}

	// Compare all properties except for problem messages.
	propertiesEquivalent := e.Kind == other.Kind &&
		e.Executable == other.Executable &&
		bytes.Equal(e.Digest, other.Digest) &&
		e.Target == other.Target
	if !propertiesEquivalent {
		return false
	}

	// Compare problem messages according to whether or not wildcard problem
	// matching is enabled. We only enable this for tests, where we can't always
	// know the exact problem message ahead of time due to variations between
	// different operating systems. Wildcard matching means that if one or both
	// of the entries has a problem message of "*", it will be considered a
	// match for the other entry's problem message.
	if !entryEqualWildcardProblemMatch {
		if e.Problem != other.Problem {
			return false
		}
	} else {
		if e.Problem != "*" && other.Problem != "*" && e.Problem != other.Problem {
			return false
		}
	}

	// If a deep comparison wasn't requested, then we're done.
	if !deep {
		return true
	}

	// Compare entry contents.
	if len(e.Contents) != len(other.Contents) {
		return false
	}
	for name, child := range e.Contents {
		otherChild, ok := other.Contents[name]
		if !ok || !child.Equal(otherChild, true) {
			return false
		}
	}

	// Done.
	return true
}

// EntryCopyBehavior indicates the type of Copy operation to perform for an
// Entry. All copy types behave the same for scalar entries - they only vary the
// behavior of directory entry copies (including phantom directories).
type EntryCopyBehavior uint8

const (
	// EntryCopyBehaviorDeep indicates that a deep copy of the entry should be
	// created.
	EntryCopyBehaviorDeep EntryCopyBehavior = iota
	// EntryCopyBehaviorDeepPreservingLeaves indicates that a deep copy of the
	// entry should be created, but that all leaf (non-directory) entry types
	// should be copied by value (i.e. by their Entry pointer) to avoid
	// allocation. This copy type can be useful if only directories in the copy
	// are going to be mutated.
	EntryCopyBehaviorDeepPreservingLeaves
	// EntryCopyBehaviorShallow indicates that a shallow copy of the entry
	// should be created.
	EntryCopyBehaviorShallow
	// EntryCopyBehaviorSlim indicates that a "slim" copy of the entry should be
	// created, which is a shallow copy that excludes the content map.
	EntryCopyBehaviorSlim
)

// Copy creates a copy of the entry using the specified copy behavior. In
// general, entries are considered immutable (by convention) and should be
// copied by pointer. However, when creating derived entries (e.g. using Apply),
// a copy operation may be necessary to create a temporarily mutable entry that
// can be modified (until returned). That is the role of this method. Although
// exported for benchmarking, there should generally be no need for code outside
// of this package to use it, except to convert a full entry to a slim entry.
func (e *Entry) Copy(behavior EntryCopyBehavior) *Entry {
	// If the entry is nil, then the copy is nil.
	if e == nil {
		return nil
	}

	// Create a slim copy.
	result := &Entry{
		Kind:       e.Kind,
		Executable: e.Executable,
		Digest:     e.Digest,
		Target:     e.Target,
		Problem:    e.Problem,
	}

	// If a slim copy was requested, then we're done.
	if behavior == EntryCopyBehaviorSlim {
		return result
	}

	// If the original entry doesn't have any contents, then return early to
	// avoid allocation of the content map.
	if len(e.Contents) == 0 {
		return result
	}

	// Copy the entry contents.
	result.Contents = make(map[string]*Entry, len(e.Contents))
	if behavior == EntryCopyBehaviorDeep {
		for name, child := range e.Contents {
			result.Contents[name] = child.Copy(EntryCopyBehaviorDeep)
		}
	} else if behavior == EntryCopyBehaviorDeepPreservingLeaves {
		for name, child := range e.Contents {
			if child.Kind == EntryKind_Directory || child.Kind == EntryKind_PhantomDirectory {
				result.Contents[name] = child.Copy(EntryCopyBehaviorDeepPreservingLeaves)
			} else {
				result.Contents[name] = child
			}
		}
	} else if behavior == EntryCopyBehaviorShallow {
		for name, child := range e.Contents {
			result.Contents[name] = child
		}
	} else {
		panic("unhandled entry copy behavior")
	}

	// Done.
	return result
}

// synchronizable returns the subtree of the entry hierarchy consisting of only
// synchronizable content. It is useful for constructing the new value of a
// change when attempting to propagate around unsychronizable content. It will
// return nil if the entry itself is unsynchronizable (which is technically the
// synchronizable subtree of the entry hierarchy in that case).
func (e *Entry) synchronizable() *Entry {
	// If the entry itself is nil, then the resulting subtree is nil.
	if e == nil {
		return nil
	}

	// If the entry itself consists of unsynchronizable content, then the
	// resulting subtree is nil.
	if !e.Kind.synchronizable() {
		return nil
	}

	// If the entry (which we know is synchronizable) is not a directory, then
	// we can just return the entry itself.
	if e.Kind != EntryKind_Directory {
		return e
	}

	// If the entry (which we know is a directory) doesn't have any contents,
	// then we can just return the entry itself.
	if len(e.Contents) == 0 {
		return e
	}

	// Create a slim copy of the entry. We only need to copy fields for
	// synchronizable entry types since we know this entry is synchronizable.
	result := &Entry{
		Kind:       e.Kind,
		Executable: e.Executable,
		Digest:     e.Digest,
		Target:     e.Target,
	}

	// Copy the entry contents. Some may not be synchronizable, in which case we
	// exclude them from the resulting map. We don't need to worry about them
	// already having been nil since nil entries aren't allowed in content maps.
	result.Contents = make(map[string]*Entry, len(e.Contents))
	for name, child := range e.Contents {
		if child = child.synchronizable(); child != nil {
			result.Contents[name] = child
		}
	}

	// Done.
	return result
}

// Problems generates a list of problems from the problematic entries contained
// within the entry hierarchy. The problems are returned in depth-first but
// non-deterministic order. Problem paths are computed assuming the entry
// represents the synchronization root.
func (e *Entry) Problems() []*Problem {
	// Create the result.
	var result []*Problem

	// Perform a walk to record problematic entries.
	e.walk("", func(path string, entry *Entry) {
		if entry != nil && entry.Kind == EntryKind_Problematic {
			result = append(result, &Problem{
				Path:  path,
				Error: entry.Problem,
			})
		}
	}, false)

	// Done.
	return result
}
