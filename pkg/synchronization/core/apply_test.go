package core

import (
	"testing"
)

// TestApply tests Apply.
func TestApply(t *testing.T) {
	// Define test cases.
	var tests = []struct {
		base          *Entry
		changes       []*Change
		expectFailure bool
		expected      *Entry
	}{
		{tN, nil, false, tN},
		{tN, []*Change{{New: tF1}}, false, tF1},
		{tN, []*Change{{New: tF1}, {New: tF2}}, false, tF2},
		{tF1, nil, false, tF1},
		{tD0, []*Change{{Path: "file", New: tF1}}, false, tD1},
		{tD0, []*Change{{Path: "missing/file", New: tF1}}, true, nil},
		{tD1, Diff(tD1, tD2), false, tD2},
		{tD0, Diff(tD0, tDM), false, tDM},
		{tDM, Diff(tDM, tD0), false, tD0},
		{nested("child", tD1), []*Change{{Path: "child/file"}}, false, nested("child", tD0)},
	}

	// Process test cases.
	for i, test := range tests {
		if result, err := Apply(test.base, test.changes); err != nil {
			if test.expectFailure {
				continue
			}
			t.Errorf("test index %d: unable to apply changes: %v", i, err)
		} else if !result.Equal(test.expected, true) {
			t.Errorf("test index %d: result did not match expected", i)
		}
	}
}
