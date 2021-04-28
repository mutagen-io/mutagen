package transport

import (
	"testing"
)

// TestProcessAttributes tests ProcessAttributes.
func TestProcessAttributes(t *testing.T) {
	if ProcessAttributes() == nil {
		t.Error("nil transport process attributes returned")
	}
}
