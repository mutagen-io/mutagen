package sync

import (
	"testing"
)

type ignoreTestValue struct {
	path     string
	expected bool
}

type ignoreTestCase struct {
	ignores []string
	tests   []ignoreTestValue
}

func (c *ignoreTestCase) run(t *testing.T) {
	// Ensure that all patterns are valid.
	for _, p := range c.ignores {
		if !ValidIgnorePattern(p) {
			t.Fatal("invalid ignore pattern detected:", p)
		}
	}

	// Create an ignorer.
	ignorer, err := newIgnorer(c.ignores)
	if err != nil {
		t.Fatal("unable to create ignorer:", err)
	}

	// Verify test values.
	for _, p := range c.tests {
		if ignorer.ignored(p.path) != p.expected {
			t.Error("ignore behavior not as expected for", p.path)
		}
	}
}

func TestNoIgnores(t *testing.T) {
	test := &ignoreTestCase{
		ignores: nil,
		tests: []ignoreTestValue{
			{"something", false},
			{"some/path", false},
		},
	}
	test.run(t)
}

func TestBasicIgnores(t *testing.T) {
	test := &ignoreTestCase{
		ignores: []string{
			"something",
			"otherthing",
			"!something",
		},
		tests: []ignoreTestValue{
			{"", false},
			{"something", false},
			{"something/other", false},
			{"otherthing", true},
			{"some/path", false},
		},
	}
	test.run(t)
}

func TestNegateOrdering(t *testing.T) {
	test := &ignoreTestCase{
		ignores: []string{
			"!something",
			"otherthing",
			"something",
		},
		tests: []ignoreTestValue{
			{"", false},
			{"something", true},
			{"something/other", false},
			{"otherthing", true},
			{"some/path", false},
		},
	}
	test.run(t)
}

func TestWildcard(t *testing.T) {
	test := &ignoreTestCase{
		ignores: []string{
			"some*",
			"!someone",
		},
		tests: []ignoreTestValue{
			{"", false},
			{"som", false},
			{"some", true},
			{"something", true},
			{"someone", false},
			{"some/path", false},
		},
	}
	test.run(t)
}

func TestPathWildcard(t *testing.T) {
	test := &ignoreTestCase{
		ignores: []string{
			"some/*",
			"some/**/*",
			"!some/other",
		},
		tests: []ignoreTestValue{
			{"", false},
			{"something", false},
			{"some", false},
			{"some/path", true},
			{"some/other", false},
			{"some/other/path", true},
		},
	}
	test.run(t)
}

func TestEmptyIgnorePatternInvalid(t *testing.T) {
	if ValidIgnorePattern("") {
		t.Fatal("empty pattern should be invalid")
	}
}

func TestInvalidPattern(t *testing.T) {
	if ValidIgnorePattern("\\") {
		t.Fatal("invalid pattern should be invalid")
	}
}

func TestInvalidPatternOnIgnorer(t *testing.T) {
	if ignorer, err := newIgnorer([]string{"\\"}); err == nil {
		t.Error("ignorer creation should fail on invalid pattern")
	} else if ignorer != nil {
		t.Error("ignorer should be nil on failed creation")
	}
}
