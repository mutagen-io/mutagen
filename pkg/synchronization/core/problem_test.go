package core

import (
	"testing"
)

// TestProblemEnsureValid tests Problem.EnsureValid.
func TestProblemEnsureValid(t *testing.T) {
	// Define test cases.
	tests := []struct {
		problem  *Problem
		expected bool
	}{
		{nil, false},
		{&Problem{}, false},
		{&Problem{Path: "/some/path"}, false},
		{&Problem{Error: "some root error"}, true},
		{&Problem{Path: "/some/path", Error: "some path error"}, true},
	}

	// Process test cases.
	for i, test := range tests {
		if err := test.problem.EnsureValid(); err == nil && !test.expected {
			t.Errorf("test index %d: problem unexpectedly classified as valid", i)
		} else if err != nil && test.expected {
			t.Errorf("test index %d: problem unexpectedly classified as invalid: %v", i, err)
		}
	}
}

// TODO: Implement TestCopyProblems.

// TODO: Implement TestSortProblems.
