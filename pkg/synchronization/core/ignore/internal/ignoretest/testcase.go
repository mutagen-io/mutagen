package ignoretest

import (
	"testing"

	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore"
)

// ignoreStatusDescription returns a human-readable ignore status description.
func ignoreStatusDescription(status ignore.IgnoreStatus) string {
	switch status {
	case ignore.IgnoreStatusNominal:
		return "nominal"
	case ignore.IgnoreStatusIgnored:
		return "ignored"
	case ignore.IgnoreStatusUnignored:
		return "unignored"
	default:
		return "unknown"
	}
}

// TestValue encodes a test operation in an TestCase.
type TestValue struct {
	// Path is the path to test.
	Path string
	// Directory indicates whether or not the path is a directory.
	Directory bool
	// ExpectedStatus is the expected ignore status.
	ExpectedStatus ignore.IgnoreStatus
	// ExpectedContinueTraversal is the expected traversal continuation status.
	ExpectedContinueTraversal bool
}

// TestCase encodes a sequence of test values for a specified set of ignore
// patterns.
type TestCase struct {
	// PatternValidator is the pattern validation callback.
	PatternValidator func(string) error
	// Constructor is the ignorer constructor callback.
	Constructor func([]string) (ignore.Ignorer, error)
	// Ignores are the ignore patterns.
	Ignores []string
	// Tests are the ignore tests to run.
	Tests []TestValue
}

// Run invokes the test with the specified test runner.
func (c *TestCase) Run(t *testing.T) {
	// Mark this runner as a helper.
	t.Helper()

	// Ensure that all patterns are valid.
	for _, pattern := range c.Ignores {
		if err := c.PatternValidator(pattern); err != nil {
			t.Fatalf("invalid ignore pattern (%s): %v", pattern, err)
		}
	}

	// Create the ignorer.
	ignorer, err := c.Constructor(c.Ignores)
	if err != nil {
		t.Fatal("unable to create ignorer:", err)
	}

	// Verify test values.
	for i, test := range c.Tests {
		status, continueTraversal := ignorer.Ignore(test.Path, test.Directory)
		if status != test.ExpectedStatus {
			t.Errorf("test index %d: ignore status (%s) not as expected (%s) for %s",
				i, ignoreStatusDescription(status), ignoreStatusDescription(test.ExpectedStatus), test.Path,
			)
		}
		if continueTraversal != test.ExpectedContinueTraversal {
			t.Errorf("test index %d: traversal continuation (%t) not as expected (%t) for %s",
				i, continueTraversal, test.ExpectedContinueTraversal, test.Path,
			)
		}
	}
}
