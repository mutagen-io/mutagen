package random

import (
	"testing"
)

// TestNew tests New.
func TestNew(t *testing.T) {
	if data, err := New(CollisionResistantLength); err != nil {
		t.Fatal("unable to create random data:", err)
	} else if len(data) != CollisionResistantLength {
		t.Error("random data did not have expected length:", len(data), "!=", CollisionResistantLength)
	}
}
