package sync

import (
	"testing"
)

func TestApplyRootSwap(t *testing.T) {
	changes := []Change{
		Change{
			Old: gorootSnapshot.get("bin"),
			New: gorootSnapshot.get("VERSION"),
		},
	}
	if result, err := Apply(gorootSnapshot.get("bin"), changes); err != nil {
		t.Fatal("unable to apply changes:", err)
	} else if !result.Equal(gorootSnapshot.get("VERSION")) {
		t.Error("mismatch after root replacement")
	}
}

func TestDiffApply(t *testing.T) {
	changes := Diff(gorootSnapshot.get("doc"), gorootSnapshot.get("src"))
	if result, err := Apply(gorootSnapshot.get("doc"), changes); err != nil {
		t.Fatal("unable to apply changes:", err)
	} else if !result.Equal(gorootSnapshot.get("src")) {
		t.Error("mismatch after diff/apply cycle")
	}
}
