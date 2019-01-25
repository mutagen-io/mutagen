package sync

import (
	"testing"
)

func TestChangeNilInvalid(t *testing.T) {
	var change *Change
	if change.EnsureValid() == nil {
		t.Error("nil change considered valid")
	}
}

func TestChangeValid(t *testing.T) {
	change := &Change{New: testSymlinkEntry}
	if err := change.EnsureValid(); err != nil {
		t.Error("valid change considered invalid:", err)
	}
}
