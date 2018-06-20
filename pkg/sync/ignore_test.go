package sync

import (
	"testing"
)

func TestIgnoreVCSModeUnmarshalTrue(t *testing.T) {
	var mode IgnoreVCSMode
	if err := mode.UnmarshalText([]byte("true")); err != nil {
		t.Fatal("unable to unmarshal text:", err)
	} else if mode != IgnoreVCSMode_IgnoreVCS {
		t.Error("unmarshalled mode does not match expected")
	}
}

func TestIgnoreVCSModeUnmarshalFalse(t *testing.T) {
	var mode IgnoreVCSMode
	if err := mode.UnmarshalText([]byte("false")); err != nil {
		t.Fatal("unable to unmarshal text:", err)
	} else if mode != IgnoreVCSMode_PropagateVCS {
		t.Error("unmarshalled mode does not match expected")
	}
}

func TestIgnoreVCSModeUnmarshalEmpty(t *testing.T) {
	var mode IgnoreVCSMode
	if mode.UnmarshalText([]byte("")) == nil {
		t.Error("empty VCS ignore mode successfully unmarshalled")
	}
}

func TestIgnoreVCSModeUnmarshalInvalid(t *testing.T) {
	var mode IgnoreVCSMode
	if mode.UnmarshalText([]byte("invalid")) == nil {
		t.Error("invalid VCS ignore mode successfully unmarshalled")
	}
}

func TestIgnoreVCSModeSupported(t *testing.T) {
	if IgnoreVCSMode_IgnoreVCSDefault.Supported() {
		t.Error("default VCS ignore mode considered supported")
	}
	if !IgnoreVCSMode_IgnoreVCS.Supported() {
		t.Error("ignore VCS mode considered unsupported")
	}
	if !IgnoreVCSMode_PropagateVCS.Supported() {
		t.Error("propagate VCS mode considered unsupported")
	}
	if (IgnoreVCSMode_PropagateVCS + 1).Supported() {
		t.Error("invalid VCS ignore mode considered supported")
	}
}

func TestIgnoreVCSModeDescription(t *testing.T) {
	if description := IgnoreVCSMode_IgnoreVCSDefault.Description(); description != "Default" {
		t.Error("default VCS ignore mode description incorrect:", description, "!=", "Default")
	}
	if description := IgnoreVCSMode_IgnoreVCS.Description(); description != "Ignore" {
		t.Error("ignore VCS mode description incorrect:", description, "!=", "Ignore")
	}
	if description := IgnoreVCSMode_PropagateVCS.Description(); description != "Propagate" {
		t.Error("propagate VCS mode description incorrect:", description, "!=", "Propagate")
	}
	if description := (IgnoreVCSMode_PropagateVCS + 1).Description(); description != "Unknown" {
		t.Error("invalid VCS ignore mode description incorrect:", description, "!=", "Unknown")
	}
}

type ignoreTestValue struct {
	path      string
	directory bool
	expected  bool
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
		if ignorer.ignored(p.path, p.directory) != p.expected {
			t.Error("ignore behavior not as expected for", p.path)
		}
	}
}

func TestIgnoreNone(t *testing.T) {
	test := &ignoreTestCase{
		ignores: nil,
		tests: []ignoreTestValue{
			{"", false, false},
			{"", true, false},
			{"something", false, false},
			{"something", true, false},
			{"some/path", false, false},
			{"some/path", true, false},
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
			{"", false, false},
			{"", true, false},
			{"something", false, false},
			{"something", true, false},
			{"subpath/something", false, false},
			{"subpath/something", true, false},
			{"otherthing", false, true},
			{"otherthing", true, true},
			{"subpath/otherthing", false, true},
			{"subpath/otherthing", true, true},
			{"random", false, false},
			{"random", true, false},
			{"subpath/random", false, false},
			{"subpath/random", true, false},
			{"somedir", false, false},
			{"somedir", true, true},
			{"subpath/somedir", false, false},
			{"subpath/somedir", true, true},
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
			{"", false, false},
			{"", true, false},
			{"run.py", false, false},
			{"run.pyc", false, true},
			{"run.pyc", true, true},
			{"subpath/run.pyd", false, true},
			{"subpath/run.pyd", true, true},
			{"run.dir", false, false},
			{"run.dir", true, false},
			{"run.dirc", false, false},
			{"run.dirc", true, true},
			{"subpath/run.dird", false, false},
			{"subpath/run.dird", true, true},
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
			{"", false, false},
			{"", true, false},
			{"abspath", false, true},
			{"abspath", true, true},
			{"subpath/abspath", false, false},
			{"subpath/abspath", true, false},
			{"absdir", false, false},
			{"absdir", true, true},
			{"subpath/absdir", false, false},
			{"subpath/absdir", true, false},
			{"name", false, true},
			{"name", true, true},
			{"subpath/name", false, false},
			{"subpath/name", true, false},
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
			{"", false, false},
			{"", true, false},
			{"something", false, false},
			{"some", false, false},
			{"some/path", false, true},
			{"some/other", false, false},
			{"some/other/path", false, true},
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
			{"", false, false},
			{"", true, false},
			{"something", false, true},
			{"something/other", false, false},
			{"otherthing", false, true},
			{"some/path", false, false},
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
			{"", false, false},
			{"", true, false},
			{"som", false, false},
			{"some", false, true},
			{"something", false, true},
			{"someone", false, false},
			{"some/path", false, false},
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
			{"", false, false},
			{"", true, false},
			{"something", false, false},
			{"some", false, false},
			{"some/path", false, true},
			{"some/other", false, false},
			{"some/other/path", false, true},
			{"subdir/some/other/path", false, false},
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
	if ignorer, err := newIgnorer([]string{"\\"}); err == nil {
		t.Error("ignorer creation should fail on invalid pattern")
	} else if ignorer != nil {
		t.Error("ignorer should be nil on failed creation")
	}
}
