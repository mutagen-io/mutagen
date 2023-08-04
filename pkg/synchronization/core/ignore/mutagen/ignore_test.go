package mutagen

import (
	"testing"

	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore/internal/ignoretest"
)

// TestCleanPreservingTrailingSlash tests that cleanPreservingTrailingSlash
// behaves as expected.
func TestCleanPreservingTrailingSlash(t *testing.T) {
	// Define test cases.
	tests := []struct {
		input    string
		expected string
	}{
		{"", "."},
		{"/", "/"},
		{"//", "//"},
		{"///", "//"},
		{"/a", "/a"},
		{" /a", " /a"},
		{" /a/", " /a/"},
		{"a/", "a/"},
		{"a//", "a/"},
		{"a", "a"},
		{" ", " "},
		{" //", " /"},
	}

	// Process test cases.
	for i, test := range tests {
		if output := cleanPreservingTrailingSlash(test.input); output != test.expected {
			t.Errorf("test index %d: output did not match expected: \"%s\" != \"%s\"", i, output, test.expected)
		}
	}
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
		{"///", false},
		{"!///", false},
		{"\\", false},

		{"some pattern", true},
		{"some/pattern", true},
		{"/some/pattern", true},
		{"/some/pattern/", true},
		{"\t \n", true},
	}

	// Process test cases.
	for i, test := range tests {
		if err := EnsurePatternValid(test.pattern); err != nil && test.expectValid {
			t.Errorf("test index %d: pattern (%s) was unexpectedly classified as invalid: %v", i, test.pattern, err)
		} else if err == nil && !test.expectValid {
			t.Errorf("test index %d: pattern (%s) was unexpectedly classified as valid", i, test.pattern)
		}
	}
}

func TestIgnoreNone(t *testing.T) {
	test := &ignoretest.TestCase{
		PatternValidator: EnsurePatternValid,
		Constructor:      NewIgnorer,
		Ignores:          nil,
		Tests: []ignoretest.TestValue{
			{"", false, ignore.IgnoreStatusNominal, false},
			{"", true, ignore.IgnoreStatusNominal, true},
			{"something", false, ignore.IgnoreStatusNominal, false},
			{"something", true, ignore.IgnoreStatusNominal, true},
			{"some/path", false, ignore.IgnoreStatusNominal, false},
			{"some/path", true, ignore.IgnoreStatusNominal, true},
		},
	}
	test.Run(t)
}

func TestIgnorerBasic(t *testing.T) {
	test := &ignoretest.TestCase{
		PatternValidator: EnsurePatternValid,
		Constructor:      NewIgnorer,
		Ignores: []string{
			"something",
			"otherthing",
			"!something",
			"somedir/",
		},
		Tests: []ignoretest.TestValue{
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
	test.Run(t)
}

func TestIgnoreGroup(t *testing.T) {
	test := &ignoretest.TestCase{
		PatternValidator: EnsurePatternValid,
		Constructor:      NewIgnorer,
		Ignores: []string{
			"*.py[cod]",
			"*.dir[cod]/",
		},
		Tests: []ignoretest.TestValue{
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
	test.Run(t)
}

func TestIgnoreRootRelative(t *testing.T) {
	test := &ignoretest.TestCase{
		PatternValidator: EnsurePatternValid,
		Constructor:      NewIgnorer,
		Ignores: []string{
			"/abspath",
			"/absdir/",
			"/name",
			"!*/**/name",
		},
		Tests: []ignoretest.TestValue{
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
	test.Run(t)
}

func TestIgnoreDoublestar(t *testing.T) {
	test := &ignoretest.TestCase{
		PatternValidator: EnsurePatternValid,
		Constructor:      NewIgnorer,
		Ignores: []string{
			"some/*",
			"some/**/*",
			"!some/other",
		},
		Tests: []ignoretest.TestValue{
			{"", false, ignore.IgnoreStatusNominal, false},
			{"", true, ignore.IgnoreStatusNominal, true},
			{"something", false, ignore.IgnoreStatusNominal, false},
			{"some", false, ignore.IgnoreStatusNominal, false},
			{"some/path", false, ignore.IgnoreStatusIgnored, false},
			{"some/other", false, ignore.IgnoreStatusUnignored, false},
			{"some/other/path", false, ignore.IgnoreStatusIgnored, false},
		},
	}
	test.Run(t)
}

func TestIgnoreNegateOrdering(t *testing.T) {
	test := &ignoretest.TestCase{
		PatternValidator: EnsurePatternValid,
		Constructor:      NewIgnorer,
		Ignores: []string{
			"!something",
			"otherthing",
			"something",
		},
		Tests: []ignoretest.TestValue{
			{"", false, ignore.IgnoreStatusNominal, false},
			{"", true, ignore.IgnoreStatusNominal, true},
			{"something", false, ignore.IgnoreStatusIgnored, false},
			{"something/other", false, ignore.IgnoreStatusNominal, false},
			{"otherthing", false, ignore.IgnoreStatusIgnored, false},
			{"some/path", false, ignore.IgnoreStatusNominal, false},
		},
	}
	test.Run(t)
}

func TestIgnoreWildcard(t *testing.T) {
	test := &ignoretest.TestCase{
		PatternValidator: EnsurePatternValid,
		Constructor:      NewIgnorer,
		Ignores: []string{
			"some*",
			"!someone",
		},
		Tests: []ignoretest.TestValue{
			{"", false, ignore.IgnoreStatusNominal, false},
			{"", true, ignore.IgnoreStatusNominal, true},
			{"som", false, ignore.IgnoreStatusNominal, false},
			{"some", false, ignore.IgnoreStatusIgnored, false},
			{"something", false, ignore.IgnoreStatusIgnored, false},
			{"someone", false, ignore.IgnoreStatusUnignored, false},
			{"some/path", false, ignore.IgnoreStatusNominal, false},
		},
	}
	test.Run(t)
}

func TestIgnorePathWildcard(t *testing.T) {
	test := &ignoretest.TestCase{
		PatternValidator: EnsurePatternValid,
		Constructor:      NewIgnorer,
		Ignores: []string{
			"some/*",
			"some/**/*",
			"!some/other",
		},
		Tests: []ignoretest.TestValue{
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
	test.Run(t)
}
