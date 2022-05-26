package synchronization

import (
	"encoding/hex"

	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
)

// Entry represents a filesystem entry.
type Entry struct {
	// Kind encodes the type of filesystem entry being represented.
	Kind core.EntryKind `json:"kind"`
	// DirectoryEntry stores fields that are only relevant to directory entries.
	// It is only non-nil if the entry is a directory.
	*DirectoryEntry
	// FileEntry stores fields that are only relevant to file entries. It is
	// only non-nil if the entry is a file.
	*FileEntry
	// SymbolicLinkEntry stores fields that are only relevant to symbolic link
	// entries. It is only non-nil if the entry is a symbolic link.
	*SymbolicLinkEntry
	// ProblematicEntry stores fields that are only relevant to problematic
	// entries. It is only non-nil if the entry is problematic content.
	*ProblematicEntry
}

// DirectoryEntry encodes fields relevant to directory entries.
type DirectoryEntry struct {
	// Contents represents a directory entry's contents.
	Contents map[string]*Entry `json:"contents"`
}

// FileEntry encodes fields relevant to file entries.
type FileEntry struct {
	// Digest represents the hash of a file entry's contents.
	Digest string `json:"digest"`
	// Executable indicates whether or not a file entry is marked as executable.
	Executable bool `json:"executable,omitempty"`
}

// SymbolicLinkEntry encodes fields relevant to symbolic link entries.
type SymbolicLinkEntry struct {
	// Target is the symbolic link target.
	Target string `json:"target"`
}

// ProblematicEntry encodes fields relevant to problematic entries.
type ProblematicEntry struct {
	// Problem indicates the relevant error for problematic content.
	Problem string `json:"problem"`
}

// newEntryFromInternalEntry creates a new entry representation from an internal
// Protocol Buffers representation. The entry must be valid.
func newEntryFromInternalEntry(entry *core.Entry) *Entry {
	// Handle the case of non-existent entries.
	if entry == nil {
		return nil
	}

	// Create the result.
	result := &Entry{Kind: entry.Kind}

	// Propagate the relevant fields.
	switch entry.Kind {
	case core.EntryKind_Directory:
		result.DirectoryEntry = &DirectoryEntry{}
		if l := len(entry.Contents); l > 0 {
			result.Contents = make(map[string]*Entry, l)
			for n, c := range entry.Contents {
				result.Contents[n] = newEntryFromInternalEntry(c)
			}
		}
	case core.EntryKind_File:
		result.FileEntry = &FileEntry{
			Digest:     hex.EncodeToString(entry.Digest),
			Executable: entry.Executable,
		}
	case core.EntryKind_SymbolicLink:
		result.SymbolicLinkEntry = &SymbolicLinkEntry{Target: entry.Target}
	case core.EntryKind_Untracked:
		// There are no fields to propagate for untracked content.
	case core.EntryKind_Problematic:
		result.ProblematicEntry = &ProblematicEntry{Problem: entry.Problem}
	default:
		panic("invalid entry kind")
	}

	// Done.
	return result
}
