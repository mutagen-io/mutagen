package sync

import (
	"testing"
)

// TestGOROOTValid rebuilds the GOROOT snapshot is valid.
func TestGOROOTValid(t *testing.T) {
	if err := gorootSnapshot.EnsureValid(); err != nil {
		t.Fatal("GOROOT invalid:", err)
	}
}
