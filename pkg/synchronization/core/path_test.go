package core

import (
	"testing"
)

// pathDirPanicFree is a wrapper around pathDir that tracks panics.
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

// TestPathDir verifies that pathDir behaves correctly.
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

// pathBasePanicFree is a wrapper around PathBase that tracks panics.
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

// TestPathBase verifies that PathBase behaves correctly.
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

// TestPathLess verifies that pathLess behaves correctly.
func TestPathLess(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		first    string
		second   string
		expected bool
	}{
		{"", "", false},
		{"a", "", false},
		{"", "a", true},
		{"a", "a", false},
		{"a/b", "b", true},
		{"b", "a/b", false},
		{"a/b", "a/b", false},
		{"a/b/c", "a", false},
		{"a/b/c", "a/b", false},
		{"a", "a/b/c", true},
		{"a/b", "a/b/c", true},
		{"a/b/c", "a/b/c", false},
		{"a/b/c", "a/d/c", true},
		{"a/b/c", "a/b/cd", true},
		{"a/b/cd", "a/b/c", false},
		{"a/b/c", "a/e/cd", true},
		{"a/e/cd", "a/b/c", false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if result := pathLess(testCase.first, testCase.second); result != testCase.expected {
			t.Errorf("pathLess result did not match expected for \"%s\" < \"%s\": %t != %t",
				testCase.first, testCase.second,
				result, testCase.expected,
			)
		}
	}
}
