package random

import (
	"testing"
)

const (
	// testLength is the length used for testing random data generation.
	testLength = 64
)

// TestNew tests New.
func TestNew(t *testing.T) {
	if data, err := New(testLength); err != nil {
		t.Fatal("unable to create random data:", err)
	} else if len(data) != testLength {
		t.Error("random data did not have expected length:", len(data), "!=", testLength)
	}
}
