package sync

import (
	"testing"
)

// pathJoinPanicFree is a wrapper around pathJoin that allows the caller to
// track panics.
func pathJoinPanicFree(base, leaf string, panicked *bool) string {
	// Track panics.
	defer func() {
		if recover() != nil {
			*panicked = true
		}
	}()

	// Invoke pathJoin.
	return pathJoin(base, leaf)
}

// TestPathJoin verifies that pathJoin behaves correctly in a number of test
// cases.
func TestPathJoin(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		base        string
		leaf        string
		expected    string
		expectPanic bool
	}{
		{"", "", "", true},
		{"a", "", "", true},
		{"", "a", "a", false},
		{"", "a/b", "a/b", false},
		{"a", "b", "a/b", false},
		{"a/b", "c/d", "a/b/c/d", false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		// Compute the result and track panics.
		var panicked bool
		if result := pathJoinPanicFree(testCase.base, testCase.leaf, &panicked); result != testCase.expected {
			t.Error("pathJoin result did not match expected:", result, "!=", testCase.expected)
		}

		// Check panic behavior.
		if panicked && !testCase.expectPanic {
			t.Error("pathJoin panicked unexpectedly")
		} else if !panicked && testCase.expectPanic {
			t.Error("pathJoin did not panic as expected")
		}
	}
}

// pathDirPanicFree is a wrapper around pathDir that allows the caller to track
// panics.
func pathDirPanicFree(path string, panicked *bool) string {
	// Track panics.
	defer func() {
		if recover() != nil {
			*panicked = true
		}
	}()

	// Invoke pathDir.
	return pathDir(path)
}

// TestPathDir verifies that pathDir behaves correctly in a number of test
// cases.
func TestPathDir(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		path        string
		expected    string
		expectPanic bool
	}{
		{"", "", true},
		{"/a", "", true},
		{"a", "", false},
		{"a/b", "a", false},
		{"a/b/c", "a/b", false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		// Compute the result and track panics.
		var panicked bool
		if result := pathDirPanicFree(testCase.path, &panicked); result != testCase.expected {
			t.Error("pathDir result did not match expected:", result, "!=", testCase.expected)
		}

		// Check panic behavior.
		if panicked && !testCase.expectPanic {
			t.Error("pathDir panicked unexpectedly")
		} else if !panicked && testCase.expectPanic {
			t.Error("pathDir did not panic as expected")
		}
	}
}

// pathBasePanicFree is a wrapper around PathBase that allows the caller to
// track panics.
func pathBasePanicFree(path string, panicked *bool) string {
	// Track panics.
	defer func() {
		if recover() != nil {
			*panicked = true
		}
	}()

	// Invoke PathBase.
	return PathBase(path)
}

// TestPathBase verifies that pathDir behaves correctly in a number of test
// cases.
func TestPathBase(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		path        string
		expected    string
		expectPanic bool
	}{
		{"", "", false},
		{"a/", "", true},
		{"a", "a", false},
		{"a/b", "b", false},
		{"a/b/c", "c", false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		// Compute the result and track panics.
		var panicked bool
		if result := pathBasePanicFree(testCase.path, &panicked); result != testCase.expected {
			t.Error("PathBase result did not match expected:", result, "!=", testCase.expected)
		}

		// Check panic behavior.
		if panicked && !testCase.expectPanic {
			t.Error("PathBase panicked unexpectedly")
		} else if !panicked && testCase.expectPanic {
			t.Error("PathBase did not panic as expected")
		}
	}
}
