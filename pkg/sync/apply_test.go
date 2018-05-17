package sync

import (
	"testing"
)

func TestApplyRootSwap(t *testing.T) {
	changes := []*Change{
		&Change{
			Old: gorootSnapshot.Contents["bin"],
			New: gorootSnapshot.Contents["VERSION"],
		},
	}
	if result, err := Apply(gorootSnapshot.Contents["bin"], changes); err != nil {
		t.Fatal("unable to apply changes:", err)
	} else if !result.Equal(gorootSnapshot.Contents["VERSION"]) {
		t.Error("mismatch after root replacement")
	}
}

func TestDiffApply(t *testing.T) {
	changes := diff("", gorootSnapshot.Contents["doc"], gorootSnapshot.Contents["src"])
	if result, err := Apply(gorootSnapshot.Contents["doc"], changes); err != nil {
		t.Fatal("unable to apply changes:", err)
	} else if !result.Equal(gorootSnapshot.Contents["src"]) {
		t.Error("mismatch after diff/apply cycle")
	}
}
