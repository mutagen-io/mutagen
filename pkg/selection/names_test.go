package selection

import (
	"testing"
)

// TestEnsureNameValid tests that EnsureNameValid behaves as expected for a
// variety of test cases.
func TestEnsureNameValid(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		name          string
		expectFailure bool
	}{
		{"", false},
		{"a", false},
		{"abc93ba1udah", false},
		{"Ac93ba1udah", false},
		{"Äbc93ba1udah", false},
		{"_", true},
		{"a9B_1", true},
		{"a b", true},
		{" ", true},
		{"a-b", true},
	}

	// Process test cases.
	for _, testCase := range testCases {
		err := EnsureNameValid(testCase.name)
		if err != nil && !testCase.expectFailure {
			t.Errorf("name (%s) failed validation unexpectedly: %v", testCase.name, err)
		} else if err == nil && testCase.expectFailure {
			t.Errorf("name (%s) passed validation unexpectedly", testCase.name)
		}
	}
}
