package numeric

import (
	"math"
	"testing"
)

// TestMaxUint64Equivalence checks that our MaxUint64 constant is equal to the
// MaxUint64 constant defined in the math package.
func TestMaxUint64Equivalence(t *testing.T) {
	if MaxUint64 != math.MaxUint64 {
		t.Error("constants not equal")
	}
}
