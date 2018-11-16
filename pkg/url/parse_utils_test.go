package url

import (
	"testing"
)

type splitExpected struct {
	split string
	raw   string
	fail  bool
}

type splitTestCase struct {
	raw        string
	at         rune
	breakOn    rune
	keepAt     bool
	checkEmpty string
	expected   splitExpected
}

func (c *splitTestCase) run(t *testing.T) {
	var split, raw, err = splitAndBreak(c.raw, c.at, c.breakOn, c.keepAt, c.checkEmpty)

	if split != c.expected.split {
		t.Error("split mismatch:", split, "!=", c.expected.split)
	}

	if raw != c.expected.raw {
		t.Error("raw mismatch:", raw, "!=", c.expected.raw)
	}

	if err != nil && !c.expected.fail {
		t.Fatal("split failed when it should have succeeded:", err)
	} else if err == nil && c.expected.fail {
		t.Fatal("split should have failed but did not")
	}
}

func TestSplit(t *testing.T) {
	var testCase = splitTestCase{
		raw:        `abc@def:ghi`,
		at:         '@',
		breakOn:    ':',
		keepAt:     false,
		checkEmpty: "",
		expected: splitExpected{
			split: "abc",
			raw:   "def:ghi",
			fail:  false,
		},
	}
	testCase.run(t)
}

func TestSplitAtAfterBreak(t *testing.T) {
	var testCase = splitTestCase{
		raw:        `abc:def@ghi`,
		at:         '@',
		breakOn:    ':',
		keepAt:     false,
		checkEmpty: "",
		expected: splitExpected{
			split: "",
			raw:   "abc:def@ghi",
			fail:  false,
		},
	}
	testCase.run(t)
}

func TestSplitWithoutCheckEmpty(t *testing.T) {
	var testCase = splitTestCase{
		raw:        `@def:ghi`,
		at:         '@',
		breakOn:    ':',
		keepAt:     false,
		checkEmpty: "",
		expected: splitExpected{
			split: "",
			raw:   "def:ghi",
			fail:  false,
		},
	}
	testCase.run(t)
}

func TestSplitWithCheckEmpty(t *testing.T) {
	var testCase = splitTestCase{
		raw:        `@def:ghi`,
		at:         '@',
		breakOn:    ':',
		keepAt:     false,
		checkEmpty: "field",
		expected: splitExpected{
			split: "",
			raw:   "def:ghi",
			fail:  true,
		},
	}
	testCase.run(t)
}

func TestSplitEscapeBreak(t *testing.T) {
	var testCase = splitTestCase{
		raw:        `abc\:def@ghi`,
		at:         '@',
		breakOn:    ':',
		keepAt:     false,
		checkEmpty: "",
		expected: splitExpected{
			split: "abc:def",
			raw:   `ghi`,
			fail:  false,
		},
	}
	testCase.run(t)
}

func TestSplitDoubleEscape(t *testing.T) {
	var testCase = splitTestCase{
		raw:        `abc\@def@\:ghi`,
		at:         '@',
		breakOn:    ':',
		keepAt:     false,
		checkEmpty: "",
		expected: splitExpected{
			split: "abc@def",
			raw:   `\:ghi`,
			fail:  false,
		},
	}
	testCase.run(t)
}

func TestSplitNoBreak(t *testing.T) {
	var testCase = splitTestCase{
		raw:        `abc:def`,
		at:         ':',
		breakOn:    '\x00',
		keepAt:     false,
		checkEmpty: "",
		expected: splitExpected{
			split: "abc",
			raw:   "def",
			fail:  false,
		},
	}
	testCase.run(t)
}

func TestSplitNoBreakAndKeepAt(t *testing.T) {
	var testCase = splitTestCase{
		raw:        `abc:def`,
		at:         ':',
		breakOn:    '\x00',
		keepAt:     true,
		checkEmpty: "",
		expected: splitExpected{
			split: "abc",
			raw:   ":def",
			fail:  false,
		},
	}
	testCase.run(t)
}

func TestSplitNoMatchWithoutBreak(t *testing.T) {
	var testCase = splitTestCase{
		raw:        `abc\:def`,
		at:         ':',
		breakOn:    '\x00',
		keepAt:     false,
		checkEmpty: "",
		expected: splitExpected{
			split: "",
			raw:   `abc\:def`,
			fail:  false,
		},
	}
	testCase.run(t)
}

func TestSplitEscapeNoMatchShouldBreak(t *testing.T) {
	var testCase = splitTestCase{
		raw:        `abc:def@test\@test2`,
		at:         '@',
		breakOn:    ':',
		keepAt:     false,
		checkEmpty: "",
		expected: splitExpected{
			split: "",
			raw:   `abc:def@test\@test2`,
			fail:  false,
		},
	}
	testCase.run(t)
}
