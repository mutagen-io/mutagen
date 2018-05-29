package sync

import (
	"testing"
)

func TestConflictNilInvalid(t *testing.T) {
	var conflict *Conflict
	if conflict.EnsureValid() == nil {
		t.Error("nil conflict considered valid")
	}
}

func TestConflictNoAlphaChangesInvalid(t *testing.T) {
	conflict := &Conflict{BetaChanges: []*Change{{New: testFileEntry}}}
	if conflict.EnsureValid() == nil {
		t.Error("conflict with no alpha changes considered valid")
	}
}

func TestConflictNoBetaChangesInvalid(t *testing.T) {
	conflict := &Conflict{AlphaChanges: []*Change{{New: testFileEntry}}}
	if conflict.EnsureValid() == nil {
		t.Error("conflict with no beta changes considered valid")
	}
}

func TestConflictInvalidAlphaChangeInvalid(t *testing.T) {
	conflict := &Conflict{
		AlphaChanges: []*Change{{}},
		BetaChanges:  []*Change{{New: testFileEntry}},
	}
	if conflict.EnsureValid() == nil {
		t.Error("conflict with invalid alpha change considered valid")
	}
}

func TestConflictInvalidBetaChangeInvalid(t *testing.T) {
	conflict := &Conflict{
		AlphaChanges: []*Change{{New: testFileEntry}},
		BetaChanges:  []*Change{{}},
	}
	if conflict.EnsureValid() == nil {
		t.Error("conflict with invalid beta change considered valid")
	}
}

func TestConflictValid(t *testing.T) {
	conflict := &Conflict{
		AlphaChanges: []*Change{{New: testFileEntry}},
		BetaChanges:  []*Change{{New: testDirectoryEntry}},
	}
	if err := conflict.EnsureValid(); err != nil {
		t.Error("valid conflict considered invalid:", err)
	}
}
