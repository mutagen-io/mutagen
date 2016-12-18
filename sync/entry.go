package sync

import (
	"bytes"
	"sort"

	"github.com/golang/protobuf/proto"
)

func (e *Entry) Find(name string) (*Entry, bool) {
	// Nil entries have no contents.
	if e == nil {
		return nil, false
	}

	// Use a binary search to find the location of the name in the contents.
	index := sort.Search(len(e.Contents), func(i int) bool {
		return e.Contents[i].Name >= name
	})

	// Check if it's a match.
	if index < len(e.Contents) && e.Contents[index].Name == name {
		return e.Contents[index].Entry, true
	}

	// No match found.
	return nil, false
}

func (e *Entry) Insert(name string, entry *Entry) {
	// Watch for nil entries.
	if e == nil {
		panic("unable to insert content into nil entry")
	}

	// Use a binary search to find the insertion index.
	insertion := sort.Search(len(e.Contents), func(i int) bool {
		return e.Contents[i].Name >= name
	})

	// Replace any existing entry with this name, otherwise insert a new one.
	if insertion < len(e.Contents) && e.Contents[insertion].Name == name {
		e.Contents[insertion].Entry = entry
	} else {
		e.Contents = append(e.Contents, nil)
		copy(e.Contents[insertion+1:], e.Contents[insertion:])
		e.Contents[insertion] = &NamedEntry{name, entry}
	}
}

func (e *Entry) Remove(name string) bool {
	// Nil entries have no contents.
	if e == nil {
		return false
	}

	// Use a binary search to find the deletion index.
	deletion := sort.Search(len(e.Contents), func(i int) bool {
		return e.Contents[i].Name >= name
	})

	// If it's a match, cut it out. Otherwise the remove operation has failed.
	if deletion < len(e.Contents) && e.Contents[deletion].Name == name {
		e.Contents = append(e.Contents[:deletion], e.Contents[deletion+1:]...)
		return true
	}
	return false
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
	for i, ec := range e.Contents {
		oc := other.Contents[i]
		if ec.Name != oc.Name || !ec.Entry.Equal(oc.Entry) {
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

	// Create the shallow copy.
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
	for _, c := range e.Contents {
		result.Contents = append(result.Contents, &NamedEntry{
			Name:  c.Name,
			Entry: c.Entry.copy(),
		})
	}

	// Done.
	return result
}

func (e *Entry) Encode() ([]byte, error) {
	// We treat nil entries as marshalling to an empty byte slice. Protocol
	// Buffers won't like this.
	if e == nil {
		return nil, nil
	}

	// Attempt to marshal.
	return proto.Marshal(e)
}

func DecodeEntry(encoded []byte) (*Entry, error) {
	// We treat an empty byte slice as indicating a nil entry, but Protocol
	// Buffers won't like this.
	if len(encoded) == 0 {
		return nil, nil
	}

	// Attempt to unmarshal.
	result := &Entry{}
	if err := proto.Unmarshal(encoded, result); err != nil {
		return nil, err
	}

	// Success.
	return result, nil
}
