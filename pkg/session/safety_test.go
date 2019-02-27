package session

import (
	"testing"
)

// TODO: Implement tests for additional functions.

// TestFilteredPathsAreSubset tests that filteredPathsAreSubset returns a
// correct assessment for a variety of test cases.
func TestFilteredPathsAreSubset(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		filteredPaths []string
		originalPaths []string
		expected      bool
	}{
		{nil, nil, true},
		{nil, []string{}, true},
		{[]string{}, []string{}, true},
		{[]string{}, nil, true},
		{[]string{"a"}, []string{"a"}, true},
		{[]string{"a"}, []string{"a", "b"}, true},
		{[]string{"b"}, []string{"a", "b"}, true},
		{[]string{"c"}, nil, false},
		{[]string{"c"}, []string{}, false},
		{[]string{"c"}, []string{"a"}, false},
		{[]string{"c"}, []string{"a", "b"}, false},
		{[]string{"a", "b"}, []string{"a", "b"}, true},
		{[]string{"b", "a"}, []string{"a", "b"}, false},
	}

	// Run test cases.
	for c, testCase := range testCases {
		if result := filteredPathsAreSubset(
			testCase.filteredPaths,
			testCase.originalPaths,
		); result != testCase.expected {
			t.Errorf(
				"result did not match expected for test case %d: %t != %t",
				c,
				result,
				testCase.expected,
			)
		}
	}
}
