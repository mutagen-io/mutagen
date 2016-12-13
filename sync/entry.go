package sync

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"hash"
	"sort"
)

func (e *Entry) digestInto(hasher hash.Hash) {
	// If the entry is nil, there is nothing to digest.
	if e == nil {
		return
	}

	// Digest the kind.
	var kindBytes [4]byte
	binary.BigEndian.PutUint32(kindBytes[:], uint32(e.Kind))
	hasher.Write(kindBytes[:])

	// Digest executability.
	var executabilityBytes [1]byte
	if e.Executable {
		executabilityBytes[0] = 1
	} else {
		executabilityBytes[0] = 0
	}
	hasher.Write(executabilityBytes[:])

	// Digest the digest (dawg).
	var digestLengthBytes [2]byte
	binary.BigEndian.PutUint16(digestLengthBytes[:], uint16(len(e.Digest)))
	hasher.Write(digestLengthBytes[:])
	if len(e.Digest) > 0 {
		hasher.Write(e.Digest)
	}

	// Digest contents count.
	var contentsCountBytes [8]byte
	binary.BigEndian.PutUint64(contentsCountBytes[:], uint64(len(e.Contents)))
	hasher.Write(contentsCountBytes[:])

	// If there aren't any contents, return to save an allocation.
	if len(e.Contents) == 0 {
		return
	}

	// Digest contents, sorting names for consistent iteration.
	names := make([]string, 0, len(e.Contents))
	for name, _ := range e.Contents {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		// Digest the name.
		var nameLengthBytes [8]byte
		binary.BigEndian.PutUint64(nameLengthBytes[:], uint64(len(name)))
		hasher.Write(nameLengthBytes[:])
		if len(name) > 0 {
			hasher.Write([]byte(name))
		}

		// Digest the content recursively.
		e.Contents[name].digestInto(hasher)
	}
}

func (e *Entry) Checksum() []byte {
	// Create a SHA-1 hasher.
	hasher := sha1.New()

	// Digest recursively.
	e.digestInto(hasher)

	// Compute the digest.
	return hasher.Sum(nil)
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
	for n, c := range e.Contents {
		if !c.Equal(other.Contents[n]) {
			return false
		}
	}

	// Success.
	return true
}

func (e *Entry) copyShallow(makeContentMap bool) *Entry {
	// If the entry is nil, the copy is nil.
	if e == nil {
		return nil
	}

	// Create an initialized content map if requested. We don't populate it, but
	// it's nicer to have its creation encapsulated in here.
	var contents map[string]*Entry
	if makeContentMap {
		contents = make(map[string]*Entry)
	}

	// Create the copy.
	return &Entry{
		Kind:       e.Kind,
		Executable: e.Executable,
		Digest:     e.Digest,
		Contents:   contents,
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
