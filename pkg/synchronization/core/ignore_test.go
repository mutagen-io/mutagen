package core

import (
	"testing"
)

type ignoreTestValue struct {
	path                 string
	directory            bool
	defaultExpected      bool
	dockerignoreExpected bool
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
	ignorer, err := newDefaultIgnorer(c.ignores)
	if err != nil {
		t.Fatal("unable to create ignorer:", err)
	}

	// Verify test values.
	for _, p := range c.tests {
		if ignored, _, err := ignorer.ignored(p.path, p.directory); err != nil {
			t.Errorf("ignore error: %s", err)
		} else if ignored != p.defaultExpected {
			t.Error("ignore behavior not as expected for", p.path)
		}
	}

	// Create an docker ignorer.
	ignorer, err = newDockerIgnorer(c.ignores)
	if err != nil {
		t.Fatal("unable to create ignorer:", err)
	}

	// Verify test values.
	for _, p := range c.tests {
		if ignored, _, err := ignorer.ignored(p.path, p.directory); err != nil {
			t.Errorf("ignore error: %s", err)
		} else if ignored != p.dockerignoreExpected {
			t.Error("dockerignore behavior not as expected for", p.path)
		}
	}
}

func TestIgnoreNone(t *testing.T) {
	test := &ignoreTestCase{
		ignores: nil,
		tests: []ignoreTestValue{
			{"", false, false, false},
			{"", true, false, false},
			{"something", false, false, false},
			{"something", true, false, false},
			{"some/path", false, false, false},
			{"some/path", true, false, false},
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
			{"", false, false, false},
			{"", true, false, false},
			{"something", false, false, false},
			{"something", true, false, false},
			{"otherthing", false, true, true},
			{"otherthing", true, true, true},
			{"random", false, false, false},
			{"random", true, false, false},
			{"somedir", false, false, true},
			{"somedir", true, true, true},
			{"subpath", true, false, false},
			{"subpath/something", false, false, false},
			{"subpath/something", true, false, false},
			{"subpath/otherthing", false, true, false},
			{"subpath/otherthing", true, true, false},
			{"subpath/random", false, false, false},
			{"subpath/random", true, false, false},
			{"subpath/somedir", false, false, false},
			{"subpath/somedir", true, true, false},
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
			{"", false, false, false},
			{"", true, false, false},
			{"run.py", false, false, false},
			{"run.pyc", false, true, true},
			{"run.pyc", true, true, true},
			{"run.dir", false, false, false},
			{"run.dir", true, false, false},
			{"run.dirc", false, false, true},
			{"run.dirc", true, true, true},
			{"subpath", true, false, false},
			{"subpath/run.pyd", false, true, false},
			{"subpath/run.pyd", true, true, false},
			{"subpath/run.dird", false, false, false},
			{"subpath/run.dird", true, true, false},
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
			{"", false, false, false},
			{"", true, false, false},
			{"abspath", false, true, true},
			{"abspath", true, true, true},
			{"absdir", false, false, true},
			{"absdir", true, true, true},
			{"name", false, true, true},
			{"name", true, true, true},
			{"subpath", true, false, false},
			{"subpath/abspath", false, false, false},
			{"subpath/abspath", true, false, false},
			{"subpath/absdir", false, false, false},
			{"subpath/absdir", true, false, false},
			{"subpath/name", false, false, false},
			{"subpath/name", true, false, false},
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
			{"", false, false, false},
			{"", true, false, false},
			{"something", false, false, false},
			{"some", false, false, false},
			{"some", true, false, false},
			{"some/path", false, true, true},
			{"some/other", false, false, false},
			{"some/other/path", false, true, false},
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
			{"", false, false, false},
			{"", true, false, false},
			{"something", false, true, true},
			{"something", true, true, true},
			{"something/other", false, false, true},
			{"otherthing", false, true, true},
			{"some", true, false, false},
			{"some/path", false, false, false},
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
			{"", false, false, false},
			{"", true, false, false},
			{"som", false, false, false},
			{"some", false, true, true},
			{"something", false, true, true},
			{"someone", false, false, false},
			{"some", true, true, true},
			{"some/path", false, false, true},
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
			{"", false, false, false},
			{"", true, false, false},
			{"something", false, false, false},
			{"some", false, false, false},
			{"some", true, false, false},
			{"some/path", false, true, true},
			{"some/other", false, false, false},
			{"some/other", true, false, false},
			{"some/other/path", false, true, false},
			{"subdir", true, false, false},
			{"subdir/some", true, false, false},
			{"subdir/some/other", true, false, false},
			{"subdir/some/other/path", false, false, false},
		},
	}
	test.run(t)
}

func TestIgnoreEmptyPatternsInvalid(t *testing.T) {
	if ValidIgnorePattern("") {
		t.Error("empty pattern should be invalid")
	}
	if ValidIgnorePattern("!") {
		t.Error("negated empty pattern should be invalid")
	}
	if ValidIgnorePattern("/") {
		t.Error("root pattern should be invalid")
	}
	if ValidIgnorePattern("!/") {
		t.Error("negated root pattern should be invalid")
	}
	if ValidIgnorePattern("//") {
		t.Error("root directory pattern should be invalid")
	}
	if ValidIgnorePattern("!//") {
		t.Error("negated root directory pattern should be invalid")
	}
}

func TestIgnoreInvalidPatternInvalid(t *testing.T) {
	if ValidIgnorePattern("\\") {
		t.Fatal("invalid pattern should be invalid")
	}
}

func TestIgnoreInvalidPatternOnIgnorerConstruction(t *testing.T) {
	if ignorer, err := newDefaultIgnorer([]string{"\\"}); err == nil {
		t.Error("ignorer creation should fail on invalid pattern")
	} else if ignorer != nil {
		t.Error("ignorer should be nil on failed creation")
	}
}
