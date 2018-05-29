package sync

import (
	"testing"
)

func TestNilConflictInvalid(t *testing.T) {
	var conflict *Conflict
	if conflict.EnsureValid() == nil {
		t.Error("nil conflict considered valid")
	}
}

func TestConflictNoAlphaChangesInvalid(t *testing.T) {
	conflict := &Conflict{
		BetaChanges: []*Change{
			{
				New: &Entry{
					Kind:       EntryKind_File,
					Executable: true,
					Digest:     []byte{0, 1, 2, 3, 4, 5, 6},
				},
			},
		},
	}
	if conflict.EnsureValid() == nil {
		t.Error("conflict with no alpha changes considered valid")
	}
}

func TestConflictNoBetaChangesInvalid(t *testing.T) {
	conflict := &Conflict{
		AlphaChanges: []*Change{
			{
				New: &Entry{
					Kind:       EntryKind_File,
					Executable: true,
					Digest:     []byte{0, 1, 2, 3, 4, 5, 6},
				},
			},
		},
	}
	if conflict.EnsureValid() == nil {
		t.Error("conflict with no beta changes considered valid")
	}
}

func TestConflictInvalidAlphaChangeInvalid(t *testing.T) {
	conflict := &Conflict{
		AlphaChanges: []*Change{
			{},
		},
		BetaChanges: []*Change{
			{
				New: &Entry{
					Kind:       EntryKind_File,
					Executable: true,
					Digest:     []byte{0, 1, 2, 3, 4, 5, 6},
				},
			},
		},
	}
	if conflict.EnsureValid() == nil {
		t.Error("conflict with invalid alpha change considered valid")
	}
}

func TestConflictInvalidBetaChangeInvalid(t *testing.T) {
	conflict := &Conflict{
		AlphaChanges: []*Change{
			{
				New: &Entry{
					Kind:       EntryKind_File,
					Executable: true,
					Digest:     []byte{0, 1, 2, 3, 4, 5, 6},
				},
			},
		},
		BetaChanges: []*Change{
			{},
		},
	}
	if conflict.EnsureValid() == nil {
		t.Error("conflict with invalid beta change considered valid")
	}
}

func TestValidConflictValid(t *testing.T) {
	conflict := &Conflict{
		AlphaChanges: []*Change{
			{
				New: &Entry{
					Kind:       EntryKind_File,
					Executable: true,
					Digest:     []byte{0, 1, 2, 3, 4, 5, 6},
				},
			},
		},
		BetaChanges: []*Change{
			{
				New: &Entry{
					Kind:       EntryKind_File,
					Executable: true,
					Digest:     []byte{0, 1, 2, 3, 4, 5, 6},
				},
			},
		},
	}
	if err := conflict.EnsureValid(); err != nil {
		t.Error("valid conflict considered invalid:", err)
	}
}
