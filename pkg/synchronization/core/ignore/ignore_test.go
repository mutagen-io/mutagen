package ignore

import (
	"testing"
)

// TestEnsurePatternValid tests that EnsurePatternValid behaves as expected.
func TestEnsurePatternValid(t *testing.T) {
	// Define test cases.
	tests := []struct {
		pattern     string
		expectValid bool
	}{
		{"", false},
		{"!", false},
		{"/", false},
		{"!/", false},
		{"//", false},
		{"!//", false},
		{"\t \n", false},
		{"some pattern", true},
		{"some/pattern", true},
		{"/some/pattern", true},
		{"/some/pattern/", true},
	}

	// Process test cases.
	for i, test := range tests {
		if err := EnsurePatternValid(test.pattern); err != nil && test.expectValid {
			t.Errorf("test index %d: pattern was unexpectedly classified as invalid: %v", i, err)
		} else if err == nil && !test.expectValid {
			t.Errorf("test index %d: pattern was unexpectedly classified as valid", i)
		}
	}
}
