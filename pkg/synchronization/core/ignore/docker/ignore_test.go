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
			{Path: "", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "something", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "something", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "some/path", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "some/path", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
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
			"ignored",
			"!ignored/subpath",
			"!ignored/subpath2/content",
		},
		Tests: []ignoretest.TestValue{
			{Path: "", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "something", Directory: false, ExpectedStatus: ignore.IgnoreStatusUnignored, ExpectedContinueTraversal: false},
			{Path: "something", Directory: true, ExpectedStatus: ignore.IgnoreStatusUnignored, ExpectedContinueTraversal: false},
			{Path: "subpath/something", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "subpath/something", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "otherthing", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "otherthing", Directory: true, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "subpath/otherthing", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "subpath/otherthing", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "random", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "random", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "subpath/random", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "subpath/random", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "somedir", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "somedir", Directory: true, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "subpath/somedir", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "subpath/somedir", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "ignored", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "ignored", Directory: true, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: true},
			{Path: "ignored/subpath2", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "ignored/subpath2", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: true},
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
			{Path: "", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "run.py", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "run.pyc", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "run.pyc", Directory: true, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "subpath/run.pyd", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "subpath/run.pyd", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "run.dir", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "run.dir", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "run.dirc", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "run.dirc", Directory: true, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "subpath/run.dird", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "subpath/run.dird", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
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
			{Path: "", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "abspath", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "abspath", Directory: true, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "subpath/abspath", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "subpath/abspath", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "absdir", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "absdir", Directory: true, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "subpath/absdir", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "subpath/absdir", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "name", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "name", Directory: true, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "subpath/name", Directory: false, ExpectedStatus: ignore.IgnoreStatusUnignored, ExpectedContinueTraversal: false},
			{Path: "subpath/name", Directory: true, ExpectedStatus: ignore.IgnoreStatusUnignored, ExpectedContinueTraversal: false},
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
			{Path: "", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "something", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "some", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "some/path", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "some/other", Directory: false, ExpectedStatus: ignore.IgnoreStatusUnignored, ExpectedContinueTraversal: false},
			{Path: "some/other/path", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
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
			{Path: "", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "something", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "something/other", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "otherthing", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "some/path", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
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
			{Path: "", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "som", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "some", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "something", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "someone", Directory: false, ExpectedStatus: ignore.IgnoreStatusUnignored, ExpectedContinueTraversal: false},
			{Path: "some/path", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
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
			{Path: "", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "", Directory: true, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "something", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "some", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
			{Path: "some/path", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "some/other", Directory: false, ExpectedStatus: ignore.IgnoreStatusUnignored, ExpectedContinueTraversal: false},
			{Path: "some/other/path", Directory: false, ExpectedStatus: ignore.IgnoreStatusIgnored, ExpectedContinueTraversal: false},
			{Path: "subdir/some/other/path", Directory: false, ExpectedStatus: ignore.IgnoreStatusNominal, ExpectedContinueTraversal: false},
		},
	}
	test.Run(t)
}
