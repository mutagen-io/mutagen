package fastpath

import (
	"testing"
)

// dirPanicFree is a wrapper around Dir that tracks panics.
func dirPanicFree(path string, panicked *bool) string {
	// Track panics.
	defer func() {
		if recover() != nil {
			*panicked = true
		}
	}()

	// Invoke Dir.
	return Dir(path)
}

// TestDir verifies that Dir behaves correctly.
func TestDir(t *testing.T) {
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
		if result := dirPanicFree(testCase.path, &panicked); result != testCase.expected {
			t.Error("Dir result did not match expected:", result, "!=", testCase.expected)
		}

		// Check panic behavior.
		if panicked && !testCase.expectPanic {
			t.Error("Dir panicked unexpectedly")
		} else if !panicked && testCase.expectPanic {
			t.Error("Dir did not panic as expected")
		}
	}
}

// basePanicFree is a wrapper around Base that tracks panics.
func basePanicFree(path string, panicked *bool) string {
	// Track panics.
	defer func() {
		if recover() != nil {
			*panicked = true
		}
	}()

	// Invoke Base.
	return Base(path)
}

// TestBase verifies that Base behaves correctly.
func TestBase(t *testing.T) {
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
		if result := basePanicFree(testCase.path, &panicked); result != testCase.expected {
			t.Error("Base result did not match expected:", result, "!=", testCase.expected)
		}

		// Check panic behavior.
		if panicked && !testCase.expectPanic {
			t.Error("Base panicked unexpectedly")
		} else if !panicked && testCase.expectPanic {
			t.Error("Base did not panic as expected")
		}
	}
}

// TestLess verifies that Less behaves correctly.
func TestLess(t *testing.T) {
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
		if result := Less(testCase.first, testCase.second); result != testCase.expected {
			t.Errorf("Less result did not match expected for \"%s\" < \"%s\": %t != %t",
				testCase.first, testCase.second,
				result, testCase.expected,
			)
		}
	}
}
