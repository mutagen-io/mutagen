package sync

import (
	"bytes"
	"sort"

	"github.com/golang/protobuf/proto"
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
	for name, entry := range e.Contents {
		otherEntry, ok := other.Contents[name]
		if !ok || !entry.Equal(otherEntry) {
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

	// If the original entry doesn't have any contents, return now to save an
	// allocation.
	if len(e.Contents) == 0 {
		return result
	}

	// Copy contents.
	result.Contents = make(map[string]*Entry)
	for name, entry := range e.Contents {
		result.Contents[name] = entry.copy()
	}

	// Done.
	return result
}

// byName provides the sort interface for OrderedEntryContent, sorting by name.
type byName []*OrderedEntryContent

func (n byName) Len() int {
	return len(n)
}

func (n byName) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n byName) Less(i, j int) bool {
	return n[i].Name < n[j].Name
}

func (e *Entry) orderedCopy() *OrderedEntry {
	// If the entry is nil, then the copy is nil.
	if e == nil {
		return nil
	}

	// Create the result.
	result := &OrderedEntry{
		Kind:       e.Kind,
		Executable: e.Executable,
		Digest:     e.Digest,
	}

	// Copy contents.
	for name, entry := range e.Contents {
		result.Contents = append(result.Contents, &OrderedEntryContent{
			Name:  name,
			Entry: entry.orderedCopy(),
		})
	}

	// Sort contents by name.
	sort.Sort(byName(result.Contents))

	// Done.
	return result
}

// Encode serializes an entry into a byte slice using Protocol Buffers
// serialization, with a few variations. First, if the entry is nil, it returns
// an empty slice but no error (something Protocol Buffers won't do). Second,
// and most importantly, it encodes entries using a canonical scheme where
// entries that are equal will have byte-for-byte equal serializations. It does
// this with a bit of a hack. Specifically, it converts the entry message into
// a message that has a wire-equivalent layout, but instead of having a content
// map (which has random iteration and hence serialization order), it has a
// slice of name-ordered content messages that will have compatible
// serialization. This is possible because Protocol Buffers encodes maps as if
// they were repeated two-element messages:
// https://developers.google.com/protocol-buffers/docs/proto3#backwards-compatibility
// In fact, it's even recommended that developers use this strategy to maintain
// backwards compatibility with Protocol Buffers implementations that don't
// support maps.
func (e *Entry) Encode() ([]byte, error) {
	// We treat nil entries as marshalling to an empty byte slice. Protocol
	// Buffers won't like this.
	if e == nil {
		return nil, nil
	}

	// Convert the entry to its ordered equivalent and serialize that.
	return proto.Marshal(e.orderedCopy())
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
