package mutagen

import (
	"testing"

	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore"
)

type ignoreTestValue struct {
	path                      string
	directory                 bool
	expectedStatus            ignore.IgnoreStatus
	expectedContinueTraversal bool
}

type ignoreTestCase struct {
	ignores []string
	tests   []ignoreTestValue
}

func (c *ignoreTestCase) run(t *testing.T) {
	// Ensure that all patterns are valid.
	for _, p := range c.ignores {
		if err := EnsurePatternValid(p); err != nil {
			t.Fatalf("invalid ignore pattern (%s): %v", p, err)
		}
	}

	// Create an ignorer.
	ignorer, err := NewIgnorer(c.ignores)
	if err != nil {
		t.Fatal("unable to create ignorer:", err)
	}

	// Verify test values.
	for _, p := range c.tests {
		status, continueTraversal := ignorer.Ignore(p.path, p.directory)
		if status != p.expectedStatus {
			t.Error("ignore status not as expected for", p.path)
		}
		if continueTraversal != p.expectedContinueTraversal {
			t.Error("ignore traversal continuation not as expected for", p.path)
		}
	}
}

func TestIgnoreNone(t *testing.T) {
	test := &ignoreTestCase{
		ignores: nil,
		tests: []ignoreTestValue{
			{"", false, ignore.IgnoreStatusNominal, false},
			{"", true, ignore.IgnoreStatusNominal, true},
			{"something", false, ignore.IgnoreStatusNominal, false},
			{"something", true, ignore.IgnoreStatusNominal, true},
			{"some/path", false, ignore.IgnoreStatusNominal, false},
			{"some/path", true, ignore.IgnoreStatusNominal, true},
		},
	}
	test.run(t)
}

func TestIgnorerBasic(t *testing.T) {
	test := &ignoreTestCase{
		ignores: []string{
			"something",
			"otherthing",
			"!something",
			"somedir/",
		},
		tests: []ignoreTestValue{
			{"", false, ignore.IgnoreStatusNominal, false},
			{"", true, ignore.IgnoreStatusNominal, true},
			{"something", false, ignore.IgnoreStatusUnignored, false},
			{"something", true, ignore.IgnoreStatusUnignored, true},
			{"subpath/something", false, ignore.IgnoreStatusUnignored, false},
			{"subpath/something", true, ignore.IgnoreStatusUnignored, true},
			{"otherthing", false, ignore.IgnoreStatusIgnored, false},
			{"otherthing", true, ignore.IgnoreStatusIgnored, false},
			{"subpath/otherthing", false, ignore.IgnoreStatusIgnored, false},
			{"subpath/otherthing", true, ignore.IgnoreStatusIgnored, false},
			{"random", false, ignore.IgnoreStatusNominal, false},
			{"random", true, ignore.IgnoreStatusNominal, true},
			{"subpath/random", false, ignore.IgnoreStatusNominal, false},
			{"subpath/random", true, ignore.IgnoreStatusNominal, true},
			{"somedir", false, ignore.IgnoreStatusNominal, false},
			{"somedir", true, ignore.IgnoreStatusIgnored, false},
			{"subpath/somedir", false, ignore.IgnoreStatusNominal, false},
			{"subpath/somedir", true, ignore.IgnoreStatusIgnored, false},
		},
	}
	test.run(t)
}

func TestIgnoreGroup(t *testing.T) {
	test := &ignoreTestCase{
		ignores: []string{
			"*.py[cod]",
			"*.dir[cod]/",
		},
		tests: []ignoreTestValue{
			{"", false, ignore.IgnoreStatusNominal, false},
			{"", true, ignore.IgnoreStatusNominal, true},
			{"run.py", false, ignore.IgnoreStatusNominal, false},
			{"run.pyc", false, ignore.IgnoreStatusIgnored, false},
			{"run.pyc", true, ignore.IgnoreStatusIgnored, false},
			{"subpath/run.pyd", false, ignore.IgnoreStatusIgnored, false},
			{"subpath/run.pyd", true, ignore.IgnoreStatusIgnored, false},
			{"run.dir", false, ignore.IgnoreStatusNominal, false},
			{"run.dir", true, ignore.IgnoreStatusNominal, true},
			{"run.dirc", false, ignore.IgnoreStatusNominal, false},
			{"run.dirc", true, ignore.IgnoreStatusIgnored, false},
			{"subpath/run.dird", false, ignore.IgnoreStatusNominal, false},
			{"subpath/run.dird", true, ignore.IgnoreStatusIgnored, false},
		},
	}
	test.run(t)
}

func TestIgnoreRootRelative(t *testing.T) {
	test := &ignoreTestCase{
		ignores: []string{
			"/abspath",
			"/absdir/",
			"/name",
			"!*/**/name",
		},
		tests: []ignoreTestValue{
			{"", false, ignore.IgnoreStatusNominal, false},
			{"", true, ignore.IgnoreStatusNominal, true},
			{"abspath", false, ignore.IgnoreStatusIgnored, false},
			{"abspath", true, ignore.IgnoreStatusIgnored, false},
			{"subpath/abspath", false, ignore.IgnoreStatusNominal, false},
			{"subpath/abspath", true, ignore.IgnoreStatusNominal, true},
			{"absdir", false, ignore.IgnoreStatusNominal, false},
			{"absdir", true, ignore.IgnoreStatusIgnored, false},
			{"subpath/absdir", false, ignore.IgnoreStatusNominal, false},
			{"subpath/absdir", true, ignore.IgnoreStatusNominal, true},
			{"name", false, ignore.IgnoreStatusIgnored, false},
			{"name", true, ignore.IgnoreStatusIgnored, false},
			{"subpath/name", false, ignore.IgnoreStatusUnignored, false},
			{"subpath/name", true, ignore.IgnoreStatusUnignored, true},
		},
	}
	test.run(t)
}

func TestIgnoreDoublestar(t *testing.T) {
	test := &ignoreTestCase{
		ignores: []string{
			"some/*",
			"some/**/*",
			"!some/other",
		},
		tests: []ignoreTestValue{
			{"", false, ignore.IgnoreStatusNominal, false},
			{"", true, ignore.IgnoreStatusNominal, true},
			{"something", false, ignore.IgnoreStatusNominal, false},
			{"some", false, ignore.IgnoreStatusNominal, false},
			{"some/path", false, ignore.IgnoreStatusIgnored, false},
			{"some/other", false, ignore.IgnoreStatusUnignored, false},
			{"some/other/path", false, ignore.IgnoreStatusIgnored, false},
		},
	}
	test.run(t)
}

func TestIgnoreNegateOrdering(t *testing.T) {
	test := &ignoreTestCase{
		ignores: []string{
			"!something",
			"otherthing",
			"something",
		},
		tests: []ignoreTestValue{
			{"", false, ignore.IgnoreStatusNominal, false},
			{"", true, ignore.IgnoreStatusNominal, true},
			{"something", false, ignore.IgnoreStatusIgnored, false},
			{"something/other", false, ignore.IgnoreStatusNominal, false},
			{"otherthing", false, ignore.IgnoreStatusIgnored, false},
			{"some/path", false, ignore.IgnoreStatusNominal, false},
		},
	}
	test.run(t)
}

func TestIgnoreWildcard(t *testing.T) {
	test := &ignoreTestCase{
		ignores: []string{
			"some*",
			"!someone",
		},
		tests: []ignoreTestValue{
			{"", false, ignore.IgnoreStatusNominal, false},
			{"", true, ignore.IgnoreStatusNominal, true},
			{"som", false, ignore.IgnoreStatusNominal, false},
			{"some", false, ignore.IgnoreStatusIgnored, false},
			{"something", false, ignore.IgnoreStatusIgnored, false},
			{"someone", false, ignore.IgnoreStatusUnignored, false},
			{"some/path", false, ignore.IgnoreStatusNominal, false},
		},
	}
	test.run(t)
}

func TestIgnorePathWildcard(t *testing.T) {
	test := &ignoreTestCase{
		ignores: []string{
			"some/*",
			"some/**/*",
			"!some/other",
		},
		tests: []ignoreTestValue{
			{"", false, ignore.IgnoreStatusNominal, false},
			{"", true, ignore.IgnoreStatusNominal, true},
			{"something", false, ignore.IgnoreStatusNominal, false},
			{"some", false, ignore.IgnoreStatusNominal, false},
			{"some/path", false, ignore.IgnoreStatusIgnored, false},
			{"some/other", false, ignore.IgnoreStatusUnignored, false},
			{"some/other/path", false, ignore.IgnoreStatusIgnored, false},
			{"subdir/some/other/path", false, ignore.IgnoreStatusNominal, false},
		},
	}
	test.run(t)
}

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
		{"\\", false},
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
