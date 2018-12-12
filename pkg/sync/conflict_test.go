package sync

import (
	"testing"
)

func TestConflictRootInvalid(t *testing.T) {
	// Defer a handler that checks for a panic.
	defer func() {
		if recover() == nil {
			t.Error("conflict root computation did not panic for invalid conflict")
		}
	}()

	// Create an invalid conflict.
	conflict := &Conflict{}

	// Attempt to compute the root.
	conflict.Root()
}

func TestConflictRootBothAtRoot(t *testing.T) {
	// Create a test conflict.
	conflict := &Conflict{
		AlphaChanges: []*Change{
			{
				Path: "",
				New:  testFile1Entry,
			},
		},
		BetaChanges: []*Change{
			{
				Path: "",
				New:  testFile2Entry,
			},
		},
	}

	// Set the expected root.
	expectedRoot := ""

	// Verify that the root is correct.
	if conflict.Root() != expectedRoot {
		t.Error("conflict root does not match expected:", conflict.Root(), "!=", expectedRoot)
	}
}

func TestConflictRootBothOneChangeAlphaAtRoot(t *testing.T) {
	// Create a test conflict.
	conflict := &Conflict{
		AlphaChanges: []*Change{
			{
				Path: "",
				Old:  testDirectory1Entry,
				New:  testFile1Entry,
			},
		},
		BetaChanges: []*Change{
			{
				Path: "subpath",
				New:  testFile1Entry,
			},
		},
	}

	// Set the expected root.
	expectedRoot := ""

	// Verify that the root is correct.
	if conflict.Root() != expectedRoot {
		t.Error("conflict root does not match expected:", conflict.Root(), "!=", expectedRoot)
	}
}

func TestConflictRootBothOneChangeBetaAtRoot(t *testing.T) {
	// Create a test conflict.
	conflict := &Conflict{
		AlphaChanges: []*Change{
			{
				Path: "subpath",
				New:  testFile1Entry,
			},
		},
		BetaChanges: []*Change{
			{
				Path: "",
				Old:  testDirectory1Entry,
				New:  testFile1Entry,
			},
		},
	}

	// Set the expected root.
	expectedRoot := ""

	// Verify that the root is correct.
	if conflict.Root() != expectedRoot {
		t.Error("conflict root does not match expected:", conflict.Root(), "!=", expectedRoot)
	}
}

func TestConflictRootBothOneChangeAlphaHigher(t *testing.T) {
	// Create a test conflict.
	conflict := &Conflict{
		AlphaChanges: []*Change{
			{
				Path: "path",
				Old:  testDirectory1Entry,
				New:  testFile1Entry,
			},
		},
		BetaChanges: []*Change{
			{
				Path: "path/subpath",
				New:  testFile1Entry,
			},
		},
	}

	// Set the expected root.
	expectedRoot := "path"

	// Verify that the root is correct.
	if conflict.Root() != expectedRoot {
		t.Error("conflict root does not match expected:", conflict.Root(), "!=", expectedRoot)
	}
}

func TestConflictRootBothOneChangeBetaHigher(t *testing.T) {
	// Create a test conflict.
	conflict := &Conflict{
		AlphaChanges: []*Change{
			{
				Path: "path/subpath",
				New:  testFile1Entry,
			},
		},
		BetaChanges: []*Change{
			{
				Path: "path",
				Old:  testDirectory1Entry,
				New:  testFile1Entry,
			},
		},
	}

	// Set the expected root.
	expectedRoot := "path"

	// Verify that the root is correct.
	if conflict.Root() != expectedRoot {
		t.Error("conflict root does not match expected:", conflict.Root(), "!=", expectedRoot)
	}
}

func TestConflictRootBetaMultipleAlphaAtRoot(t *testing.T) {
	// Create a test conflict.
	conflict := &Conflict{
		AlphaChanges: []*Change{
			{
				Path: "",
				Old:  testDirectory1Entry,
				New:  testFile1Entry,
			},
		},
		BetaChanges: []*Change{
			{
				Path: "subpath",
				New:  testFile1Entry,
			},
			{
				Path: "subpath2",
				New:  testFile2Entry,
			},
		},
	}

	// Set the expected root.
	expectedRoot := ""

	// Verify that the root is correct.
	if conflict.Root() != expectedRoot {
		t.Error("conflict root does not match expected:", conflict.Root(), "!=", expectedRoot)
	}
}

func TestConflictRootAlphaMultipleBetaAtRoot(t *testing.T) {
	// Create a test conflict.
	conflict := &Conflict{
		AlphaChanges: []*Change{
			{
				Path: "subpath",
				New:  testFile1Entry,
			},
			{
				Path: "subpath2",
				New:  testFile2Entry,
			},
		},
		BetaChanges: []*Change{
			{
				Path: "",
				Old:  testDirectory1Entry,
				New:  testFile1Entry,
			},
		},
	}

	// Set the expected root.
	expectedRoot := ""

	// Verify that the root is correct.
	if conflict.Root() != expectedRoot {
		t.Error("conflict root does not match expected:", conflict.Root(), "!=", expectedRoot)
	}
}

func TestConflictRootBetaMultipleAlphaHigher(t *testing.T) {
	// Create a test conflict.
	conflict := &Conflict{
		AlphaChanges: []*Change{
			{
				Path: "path",
				Old:  testDirectory1Entry,
				New:  testFile1Entry,
			},
		},
		BetaChanges: []*Change{
			{
				Path: "path/subpath",
				New:  testFile1Entry,
			},
			{
				Path: "path/subpath2",
				New:  testFile2Entry,
			},
		},
	}

	// Set the expected root.
	expectedRoot := "path"

	// Verify that the root is correct.
	if conflict.Root() != expectedRoot {
		t.Error("conflict root does not match expected:", conflict.Root(), "!=", expectedRoot)
	}
}

func TestConflictRootAlphaMultipleBetaHigher(t *testing.T) {
	// Create a test conflict.
	conflict := &Conflict{
		AlphaChanges: []*Change{
			{
				Path: "path/subpath",
				New:  testFile1Entry,
			},
			{
				Path: "path/subpath2",
				New:  testFile2Entry,
			},
		},
		BetaChanges: []*Change{
			{
				Path: "path",
				Old:  testDirectory1Entry,
				New:  testFile1Entry,
			},
		},
	}

	// Set the expected root.
	expectedRoot := "path"

	// Verify that the root is correct.
	if conflict.Root() != expectedRoot {
		t.Error("conflict root does not match expected:", conflict.Root(), "!=", expectedRoot)
	}
}

func TestConflictNilInvalid(t *testing.T) {
	var conflict *Conflict
	if conflict.EnsureValid() == nil {
		t.Error("nil conflict considered valid")
	}
}

func TestConflictNoAlphaChangesInvalid(t *testing.T) {
	conflict := &Conflict{BetaChanges: []*Change{{New: testFile1Entry}}}
	if conflict.EnsureValid() == nil {
		t.Error("conflict with no alpha changes considered valid")
	}
}

func TestConflictNoBetaChangesInvalid(t *testing.T) {
	conflict := &Conflict{AlphaChanges: []*Change{{New: testFile1Entry}}}
	if conflict.EnsureValid() == nil {
		t.Error("conflict with no beta changes considered valid")
	}
}

func TestConflictInvalidAlphaChangeInvalid(t *testing.T) {
	conflict := &Conflict{
		AlphaChanges: []*Change{{}},
		BetaChanges:  []*Change{{New: testFile1Entry}},
	}
	if conflict.EnsureValid() == nil {
		t.Error("conflict with invalid alpha change considered valid")
	}
}

func TestConflictInvalidBetaChangeInvalid(t *testing.T) {
	conflict := &Conflict{
		AlphaChanges: []*Change{{New: testFile1Entry}},
		BetaChanges:  []*Change{{}},
	}
	if conflict.EnsureValid() == nil {
		t.Error("conflict with invalid beta change considered valid")
	}
}

func TestConflictValid(t *testing.T) {
	conflict := &Conflict{
		AlphaChanges: []*Change{{New: testFile1Entry}},
		BetaChanges:  []*Change{{New: testDirectory1Entry}},
	}
	if err := conflict.EnsureValid(); err != nil {
		t.Error("valid conflict considered invalid:", err)
	}
}
