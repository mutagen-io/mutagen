package sync

import (
	"testing"
)

func TestNilChangeInvalid(t *testing.T) {
	var change *Change
	if change.EnsureValid() == nil {
		t.Error("nil change considered valid")
	}
}

func TestChangeBothNilInvalid(t *testing.T) {
	change := &Change{}
	if change.EnsureValid() == nil {
		t.Error("change with both entries nil considered valid")
	}
}

func TestChangeBothSameInvalid(t *testing.T) {
	change := &Change{
		Old: &Entry{
			Kind:       EntryKind_File,
			Executable: true,
			Digest:     []byte{0, 1, 2, 3, 4, 5, 6},
		},
		New: &Entry{
			Kind:       EntryKind_File,
			Executable: true,
			Digest:     []byte{0, 1, 2, 3, 4, 5, 6},
		},
	}
	if change.EnsureValid() == nil {
		t.Error("change with duplicate considered valid")
	}
}

func TestValidChangeValid(t *testing.T) {
	change := &Change{New: &Entry{
		Kind:       EntryKind_File,
		Executable: true,
		Digest:     []byte{0, 1, 2, 3, 4, 5, 6},
	}}
	if err := change.EnsureValid(); err != nil {
		t.Error("valid change considered invalid:", err)
	}
}
