package docker

import (
	"testing"

	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore/internal/ignoretest"
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
		{"///", false},
		{"!///", false},
		{"\\", false},
		{"\t \n", false},

		{"some pattern", true},
		{"some/pattern", true},
		{"/some/pattern", true},
		{"/some/pattern/", true},
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
			{"", true, ignore.IgnoreStatusNominal, false},
			{"something", false, ignore.IgnoreStatusNominal, false},
			{"something", true, ignore.IgnoreStatusNominal, false},
			{"some/path", false, ignore.IgnoreStatusNominal, false},
			{"some/path", true, ignore.IgnoreStatusNominal, false},
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
			{"", true, ignore.IgnoreStatusNominal, false},
			{"something", false, ignore.IgnoreStatusUnignored, false},
			{"something", true, ignore.IgnoreStatusUnignored, true},
			{"subpath/something", false, ignore.IgnoreStatusNominal, false},
			{"subpath/something", true, ignore.IgnoreStatusNominal, false},
			{"otherthing", false, ignore.IgnoreStatusIgnored, false},
			{"otherthing", true, ignore.IgnoreStatusIgnored, false},
			{"subpath/otherthing", false, ignore.IgnoreStatusNominal, false},
			{"subpath/otherthing", true, ignore.IgnoreStatusNominal, false},
			{"random", false, ignore.IgnoreStatusNominal, false},
			{"random", true, ignore.IgnoreStatusNominal, false},
			{"subpath/random", false, ignore.IgnoreStatusNominal, false},
			{"subpath/random", true, ignore.IgnoreStatusNominal, false},
			{"somedir", false, ignore.IgnoreStatusIgnored, false},
			{"somedir", true, ignore.IgnoreStatusIgnored, false},
			{"subpath/somedir", false, ignore.IgnoreStatusNominal, false},
			{"subpath/somedir", true, ignore.IgnoreStatusNominal, false},
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
			{"", true, ignore.IgnoreStatusNominal, false},
			{"run.py", false, ignore.IgnoreStatusNominal, false},
			{"run.pyc", false, ignore.IgnoreStatusIgnored, false},
			{"run.pyc", true, ignore.IgnoreStatusIgnored, false},
			{"subpath/run.pyd", false, ignore.IgnoreStatusNominal, false},
			{"subpath/run.pyd", true, ignore.IgnoreStatusNominal, false},
			{"run.dir", false, ignore.IgnoreStatusNominal, false},
			{"run.dir", true, ignore.IgnoreStatusNominal, false},
			{"run.dirc", false, ignore.IgnoreStatusIgnored, false},
			{"run.dirc", true, ignore.IgnoreStatusIgnored, false},
			{"subpath/run.dird", false, ignore.IgnoreStatusNominal, false},
			{"subpath/run.dird", true, ignore.IgnoreStatusNominal, false},
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
			{"", true, ignore.IgnoreStatusNominal, false},
			{"abspath", false, ignore.IgnoreStatusIgnored, false},
			{"abspath", true, ignore.IgnoreStatusIgnored, false},
			{"subpath/abspath", false, ignore.IgnoreStatusNominal, false},
			{"subpath/abspath", true, ignore.IgnoreStatusNominal, false},
			{"absdir", false, ignore.IgnoreStatusIgnored, false},
			{"absdir", true, ignore.IgnoreStatusIgnored, false},
			{"subpath/absdir", false, ignore.IgnoreStatusNominal, false},
			{"subpath/absdir", true, ignore.IgnoreStatusNominal, false},
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
			{"", true, ignore.IgnoreStatusNominal, false},
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
			{"", true, ignore.IgnoreStatusNominal, false},
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
			{"", true, ignore.IgnoreStatusNominal, false},
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
			{"", true, ignore.IgnoreStatusNominal, false},
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
