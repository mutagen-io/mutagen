package core

import (
	"testing"
)

// TestConflictEnsureValid tests Conflict.EnsureValid.
func TestConflictEnsureValid(t *testing.T) {
	// Define test cases.
	tests := []struct {
		conflict *Conflict
		expected bool
	}{
		{nil, false},
		{&Conflict{BetaChanges: []*Change{{New: tF1}}}, false},
		{&Conflict{AlphaChanges: []*Change{{New: tF1}}}, false},
		{
			&Conflict{
				AlphaChanges: []*Change{nil},
				BetaChanges:  []*Change{{New: tF1}},
			},
			false,
		},
		{
			&Conflict{
				AlphaChanges: []*Change{{New: tF1}},
				BetaChanges:  []*Change{nil},
			},
			false,
		},
		{
			&Conflict{
				AlphaChanges: []*Change{{New: tF1}},
				BetaChanges:  []*Change{{New: tD2}},
			},
			true,
		},
		{
			&Conflict{
				AlphaChanges: []*Change{{New: tF1}},
				BetaChanges:  []*Change{{New: tDU}},
			},
			true,
		},
		{
			&Conflict{
				AlphaChanges: []*Change{{New: tDP1}},
				BetaChanges:  []*Change{{New: tF1}},
			},
			true,
		},
	}

	// Process test cases.
	for i, test := range tests {
		if err := test.conflict.EnsureValid(); err == nil && !test.expected {
			t.Errorf("test index %d: conflict incorrectly classified as valid", i)
		} else if err != nil && test.expected {
			t.Errorf("test index %d: conflict incorrectly classified as invalid: %v", i, err)
		}
	}
}

// TestConflictSlim tests Conflict.Slim.
func TestConflictSlim(t *testing.T) {
	// Create a test conflict.
	conflict := &Conflict{
		AlphaChanges: []*Change{
			{
				Path: "",
				New:  tF1,
			},
		},
		BetaChanges: []*Change{
			{
				Path: "",
				New:  tD2,
			},
		},
	}

	// Create a slim copy.
	slim := conflict.Slim()

	// Check validity.
	if err := slim.EnsureValid(); err != nil {
		t.Fatal("slim copy of conflict is invalid:", err)
	}

	// Check alpha changes.
	if len(slim.AlphaChanges) != 1 {
		t.Error("slim copy of conflict has incorrect number of alpha changes")
	} else if !slim.AlphaChanges[0].New.Equal(tF1, true) {
		t.Error("slim copy of conflict has incorrect alpha changes")
	}

	// Check beta changes.
	if len(slim.BetaChanges) != 1 {
		t.Error("slim copy of conflict has incorrect number of beta changes")
	} else if !slim.BetaChanges[0].New.Equal(tD0, true) {
		t.Error("slim copy of conflict has incorrect beta changes")
	}
}

// TODO: Implement TestCopyConflicts.

// TODO: Implement TestSortConflicts.
