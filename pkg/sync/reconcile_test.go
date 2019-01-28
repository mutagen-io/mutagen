package sync

import (
	"testing"
)

// changeListsEqual determines whether or not two lists of changes are
// equivalent. The change lists do not need to be in the same order, but they do
// need to be structurally equivalent - i.e. not composed differently.
func changeListsEqual(actualChanges, expectedChanges []*Change) bool {
	// Verify that the number of changes is the same in each case.
	if len(actualChanges) != len(expectedChanges) {
		return false
	}

	// Index expected changes by path, because ordering is not guaranteed.
	pathToExpectedChange := make(map[string]*Change, len(expectedChanges))
	for _, expected := range expectedChanges {
		pathToExpectedChange[expected.Path] = expected
	}

	// Verify that they are equal.
	for _, actual := range actualChanges {
		// Look for the corresponding expected change. This also validates path
		// equivalence.
		expected, ok := pathToExpectedChange[actual.Path]
		if !ok {
			return false
		}

		// Verify that the old values match.
		if !actual.Old.Equal(expected.Old) {
			return false
		}

		// Verify that the new values match.
		if !actual.New.Equal(expected.New) {
			return false
		}
	}

	// At this point, the changes lists must be equivalent.
	return true
}

// conflictListsEqual determines whether or not two lists of conflicts are
// equivalent. The conflict lists do not need to be in the same order.
func conflictListsEqual(actualConflicts, expectedConflicts []*Conflict) bool {
	// Verify that the number of conflicts is the same in each case.
	if len(actualConflicts) != len(expectedConflicts) {
		return false
	}

	// Index expected conflicts by root path, because ordering is not
	// guaranteed.
	pathToExpectedConflict := make(map[string]*Conflict, len(expectedConflicts))
	for _, expected := range expectedConflicts {
		pathToExpectedConflict[expected.Root()] = expected
	}

	// Verify that they are equal.
	for _, actual := range actualConflicts {
		// Look for the corresponding expected change. This also validates
		// conflict root equivalence.
		expected, ok := pathToExpectedConflict[actual.Root()]
		if !ok {
			return false
		}

		// Verify that alpha changes are equal.
		if !changeListsEqual(actual.AlphaChanges, expected.AlphaChanges) {
			return false
		}

		// Verify that beta changes are equal.
		if !changeListsEqual(actual.BetaChanges, expected.BetaChanges) {
			return false
		}
	}

	// At this point, the changes lists must be equivalent.
	return true
}

// reconcileTestCase is a utility type for reconciliation tests.
type reconcileTestCase struct {
	// ancestor is the ancestor contents for reconciliation.
	ancestor *Entry
	// alpha is the alpha contents for reconciliation.
	alpha *Entry
	// beta is the beta contents for reconciliation.
	beta *Entry
	// synchronizationModes are the synchronization modes for which the test
	// case should apply.
	synchronizationModes []SynchronizationMode
	// expectedAncestorChanges are the expected ancestor changes.
	expectedAncestorChanges []*Change
	// expectedAlphaChanges are the expected alpha changes.
	expectedAlphaChanges []*Change
	// expectedBetaChanges are the expected beta changes.
	expectedBetaChanges []*Change
	// expectedConflicts are the expected conflicts.
	expectedConflicts []*Conflict
}

// run invokes the test case in the specified testing context.
func (c *reconcileTestCase) run(t *testing.T) {
	// Mark this as a helper function.
	t.Helper()

	// Run in each of the specified conflict resolution modes.
	for _, synchronizationMode := range c.synchronizationModes {
		// Perform reconciliation.
		ancestorChanges, alphaChanges, betaChanges, conflicts := Reconcile(
			c.ancestor, c.alpha, c.beta,
			synchronizationMode,
		)

		// Check that ancestor changes are what we expect.
		if !changeListsEqual(ancestorChanges, c.expectedAncestorChanges) {
			t.Error(
				"ancestor changes do not match expected:",
				ancestorChanges, "!=", c.expectedAncestorChanges,
				"using", synchronizationMode,
			)
		}

		// Check that alpha changes are what we expect.
		if !changeListsEqual(alphaChanges, c.expectedAlphaChanges) {
			t.Error(
				"alpha changes do not match expected:",
				alphaChanges, "!=", c.expectedAlphaChanges,
				"using", synchronizationMode,
			)
		}

		// Check that beta changes are what we expect.
		if !changeListsEqual(betaChanges, c.expectedBetaChanges) {
			t.Error(
				"beta changes do not match expected:",
				betaChanges, "!=", c.expectedBetaChanges,
				"using", synchronizationMode,
			)
		}

		// Check that conflicts are what we expect.
		if !conflictListsEqual(conflicts, c.expectedConflicts) {
			t.Error(
				"conflicts do not match expected:",
				conflicts, "!=", c.expectedConflicts,
				"using", synchronizationMode,
			)
		}
	}
}

func TestNonDeletionChangesOnly(t *testing.T) {
	changes := []*Change{
		{
			Path: "file",
			New:  testFile1Entry,
		},
		{
			Path: "directory",
			Old:  testDirectory1Entry,
		},
	}
	nonDeletionChanges := nonDeletionChangesOnly(changes)
	if len(nonDeletionChanges) != 1 {
		t.Fatal("more non-deletion changes than expected")
	} else if nonDeletionChanges[0].Path != "file" {
		t.Fatal("non-deletion change has unexpected path")
	}
}

func TestReconcileAllNil(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: nil,
		alpha:    nil,
		beta:     nil,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWaySafe,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges:     nil,
		expectedConflicts:       nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileDirectoryNothingChanged(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testDirectory1Entry,
		alpha:    testDirectory1Entry,
		beta:     testDirectory1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWaySafe,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges:     nil,
		expectedConflicts:       nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileFileNothingChanged(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testFile1Entry,
		alpha:    testFile1Entry,
		beta:     testFile1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWaySafe,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges:     nil,
		expectedConflicts:       nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaModifiedRoot(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testFile1Entry,
		alpha:    testFile2Entry,
		beta:     testFile1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWaySafe,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges: []*Change{
			{Old: testFile1Entry, New: testFile2Entry},
		},
		expectedConflicts: nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBetaModifiedRootBidirectional(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testFile1Entry,
		alpha:    testFile1Entry,
		beta:     testFile2Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges: []*Change{
			{Old: testFile1Entry, New: testFile2Entry},
		},
		expectedBetaChanges: nil,
		expectedConflicts:   nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBetaModifiedRootOneWaySafe(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testFile1Entry,
		alpha:    testFile1Entry,
		beta:     testFile2Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeOneWaySafe,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges:     nil,
		expectedConflicts: []*Conflict{
			{
				AlphaChanges: []*Change{
					{
						Old: testFile1Entry,
						New: testFile1Entry,
					},
				},
				BetaChanges: []*Change{
					{
						Old: testFile1Entry,
						New: testFile2Entry,
					},
				},
			},
		},
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBetaModifiedRootOneWayReplica(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testFile1Entry,
		alpha:    testFile1Entry,
		beta:     testFile2Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges: []*Change{
			{Old: testFile2Entry, New: testFile1Entry},
		},
		expectedConflicts: nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaDeletedRoot(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testFile1Entry,
		alpha:    nil,
		beta:     testFile1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWaySafe,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges: []*Change{
			{Old: testFile1Entry},
		},
		expectedConflicts: nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBetaDeletedRootBidirectional(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testFile1Entry,
		alpha:    testFile1Entry,
		beta:     nil,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges: []*Change{
			{Old: testFile1Entry},
		},
		expectedBetaChanges: nil,
		expectedConflicts:   nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBetaDeletedRootUnidirectional(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testFile1Entry,
		alpha:    testFile1Entry,
		beta:     nil,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeOneWaySafe,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges: []*Change{
			{New: testFile1Entry},
		},
		expectedConflicts: nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBothDeletedRoot(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testFile1Entry,
		alpha:    nil,
		beta:     nil,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWaySafe,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: []*Change{
			{},
		},
		expectedAlphaChanges: nil,
		expectedBetaChanges:  nil,
		expectedConflicts:    nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaCreatedRoot(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: nil,
		alpha:    testFile1Entry,
		beta:     nil,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWaySafe,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges: []*Change{
			{New: testFile1Entry},
		},
		expectedConflicts: nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBetaCreatedRootBidirectional(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: nil,
		alpha:    nil,
		beta:     testFile1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges: []*Change{
			{New: testFile1Entry},
		},
		expectedBetaChanges: nil,
		expectedConflicts:   nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBetaCreatedRootOneWaySafe(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: nil,
		alpha:    nil,
		beta:     testFile1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeOneWaySafe,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges:     nil,
		expectedConflicts:       nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBetaCreatedRootOneWayReplica(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: nil,
		alpha:    nil,
		beta:     testFile1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges: []*Change{
			{Old: testFile1Entry},
		},
		expectedConflicts: nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBothCreatedSameFile(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: nil,
		alpha:    testFile1Entry,
		beta:     testFile1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWaySafe,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: []*Change{
			{New: testFile1Entry},
		},
		expectedAlphaChanges: nil,
		expectedBetaChanges:  nil,
		expectedConflicts:    nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBothCreatedSameDirectory(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: nil,
		alpha:    testDirectory1Entry,
		beta:     testDirectory1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWaySafe,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: testDecomposeEntry("", testDirectory1Entry, true),
		expectedAlphaChanges:    nil,
		expectedBetaChanges:     nil,
		expectedConflicts:       nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBothCreatedPartiallyMatchingContentsTwoWaySafe(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: &Entry{},
		alpha: &Entry{
			Contents: map[string]*Entry{
				"same":      testDirectory1Entry,
				"alpha":     testFile1Entry,
				"different": testFile1Entry,
			},
		},
		beta: &Entry{
			Contents: map[string]*Entry{
				"same":      testDirectory1Entry,
				"beta":      testFile2Entry,
				"different": testDirectory3Entry,
			},
		},
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
		},
		expectedAncestorChanges: testDecomposeEntry("same", testDirectory1Entry, true),
		expectedAlphaChanges: []*Change{
			{Path: "beta", New: testFile2Entry},
		},
		expectedBetaChanges: []*Change{
			{Path: "alpha", New: testFile1Entry},
		},
		expectedConflicts: []*Conflict{
			{
				AlphaChanges: []*Change{
					{
						Path: "different",
						New:  testFile1Entry,
					},
				},
				BetaChanges: []*Change{
					{
						Path: "different",
						New:  testDirectory3Entry,
					},
				},
			},
		},
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBothCreatedPartiallyMatchingContentsTwoWayResolved(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: &Entry{},
		alpha: &Entry{
			Contents: map[string]*Entry{
				"same":      testDirectory1Entry,
				"alpha":     testFile1Entry,
				"different": testFile1Entry,
			},
		},
		beta: &Entry{
			Contents: map[string]*Entry{
				"same":      testDirectory1Entry,
				"beta":      testFile2Entry,
				"different": testDirectory3Entry,
			},
		},
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWayResolved,
		},
		expectedAncestorChanges: testDecomposeEntry("same", testDirectory1Entry, true),
		expectedAlphaChanges: []*Change{
			{Path: "beta", New: testFile2Entry},
		},
		expectedBetaChanges: []*Change{
			{Path: "alpha", New: testFile1Entry},
			{Path: "different", Old: testDirectory3Entry, New: testFile1Entry},
		},
		expectedConflicts: nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBothCreatedPartiallyMatchingContentsOneWaySafe(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: &Entry{},
		alpha: &Entry{
			Contents: map[string]*Entry{
				"same":      testDirectory1Entry,
				"alpha":     testFile1Entry,
				"different": testFile1Entry,
			},
		},
		beta: &Entry{
			Contents: map[string]*Entry{
				"same":      testDirectory1Entry,
				"beta":      testFile2Entry,
				"different": testDirectory3Entry,
			},
		},
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeOneWaySafe,
		},
		expectedAncestorChanges: testDecomposeEntry("same", testDirectory1Entry, true),
		expectedAlphaChanges:    nil,
		expectedBetaChanges: []*Change{
			{Path: "alpha", New: testFile1Entry},
		},
		expectedConflicts: []*Conflict{
			{
				AlphaChanges: []*Change{
					{
						Path: "different",
						New:  testFile1Entry,
					},
				},
				BetaChanges: []*Change{
					{
						Path: "different",
						New:  testDirectory3Entry,
					},
				},
			},
		},
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBothCreatedPartiallyMatchingContentsOneWayReplica(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: &Entry{},
		alpha: &Entry{
			Contents: map[string]*Entry{
				"same":      testDirectory1Entry,
				"alpha":     testFile1Entry,
				"different": testFile1Entry,
			},
		},
		beta: &Entry{
			Contents: map[string]*Entry{
				"same":      testDirectory1Entry,
				"beta":      testFile2Entry,
				"different": testDirectory3Entry,
			},
		},
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: testDecomposeEntry("same", testDirectory1Entry, true),
		expectedAlphaChanges:    nil,
		expectedBetaChanges: []*Change{
			{Path: "alpha", New: testFile1Entry},
			{Path: "beta", Old: testFile2Entry},
			{Path: "different", Old: testDirectory3Entry, New: testFile1Entry},
		},
		expectedConflicts: nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBothCreatedDifferentTypesSafe(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: nil,
		alpha:    testDirectory1Entry,
		beta:     testFile1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeOneWaySafe,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges:     nil,
		expectedConflicts: []*Conflict{
			{
				AlphaChanges: []*Change{
					{New: testDirectory1Entry},
				},
				BetaChanges: []*Change{
					{New: testFile1Entry},
				},
			},
		},
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBothCreatedDifferentTypesOverwrite(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: nil,
		alpha:    testDirectory1Entry,
		beta:     testFile1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges: []*Change{
			{
				Old: testFile1Entry,
				New: testDirectory1Entry,
			},
		},
		expectedConflicts: nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaDeletedRootBetaCreatedFileTwoWaySafe(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testDirectory1Entry,
		alpha:    nil,
		beta:     testFile1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges: []*Change{
			{New: testFile1Entry},
		},
		expectedBetaChanges: nil,
		expectedConflicts:   nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaDeletedRootBetaCreatedFileUnsafe(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testDirectory1Entry,
		alpha:    nil,
		beta:     testFile1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges: []*Change{
			{Old: testFile1Entry},
		},
		expectedConflicts: nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaDeletedRootBetaCreatedFileOneWaySafe(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testDirectory1Entry,
		alpha:    nil,
		beta:     testFile1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeOneWaySafe,
		},
		expectedAncestorChanges: []*Change{
			{},
		},
		expectedAlphaChanges: nil,
		expectedBetaChanges:  nil,
		expectedConflicts:    nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaCreatedFileBetaDeletedRoot(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testDirectory1Entry,
		alpha:    testFile1Entry,
		beta:     nil,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWaySafe,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges: []*Change{
			{New: testFile1Entry},
		},
		expectedConflicts: nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaDeletedRootBetaCreatedDirectoryTwoWaySafe(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testFile1Entry,
		alpha:    nil,
		beta:     testDirectory1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges: []*Change{
			{New: testDirectory1Entry},
		},
		expectedBetaChanges: nil,
		expectedConflicts:   nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaDeletedRootBetaCreatedDirectoryUnsafe(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testFile1Entry,
		alpha:    nil,
		beta:     testDirectory1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges: []*Change{
			{Old: testDirectory1Entry},
		},
		expectedConflicts: nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaDeletedRootBetaCreatedDirectoryOneWaySafe(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testFile1Entry,
		alpha:    nil,
		beta:     testDirectory1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeOneWaySafe,
		},
		expectedAncestorChanges: []*Change{
			{},
		},
		expectedAlphaChanges: nil,
		expectedBetaChanges:  nil,
		expectedConflicts:    nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaCreatedDirectoryBetaDeletedRootNonBetaWinsAll(t *testing.T) {
	// Set up the test case.
	testCase := reconcileTestCase{
		ancestor: testFile1Entry,
		alpha:    testDirectory1Entry,
		beta:     nil,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWaySafe,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges: []*Change{
			{New: testDirectory1Entry},
		},
		expectedConflicts: nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaPartiallyDeletedDirectory(t *testing.T) {
	// Set up the test case. Worth noting here is that testDirectory3Entry is a
	// subtree of testDirectory2Entry.
	testCase := reconcileTestCase{
		ancestor: testDirectory2Entry,
		alpha:    testDirectory3Entry,
		beta:     testDirectory2Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWaySafe,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges:     diff("", testDirectory2Entry, testDirectory3Entry),
		expectedConflicts:       nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBetaPartiallyDeletedDirectoryBidirectional(t *testing.T) {
	// Set up the test case. Worth noting here is that testDirectory3Entry is a
	// subtree of testDirectory2Entry.
	testCase := reconcileTestCase{
		ancestor: testDirectory2Entry,
		alpha:    testDirectory2Entry,
		beta:     testDirectory3Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    diff("", testDirectory2Entry, testDirectory3Entry),
		expectedBetaChanges:     nil,
		expectedConflicts:       nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileBetaPartiallyDeletedDirectoryUnidirectional(t *testing.T) {
	// Set up the test case. Worth noting here is that testDirectory3Entry is a
	// subtree of testDirectory2Entry.
	testCase := reconcileTestCase{
		ancestor: testDirectory2Entry,
		alpha:    testDirectory2Entry,
		beta:     testDirectory3Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeOneWaySafe,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges:     diff("", testDirectory3Entry, testDirectory2Entry),
		expectedConflicts:       nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaReplacedDirectoryBetaPartiallyDeletedDirectory(t *testing.T) {
	// Set up the test case. Worth noting here is that testDirectory3Entry is a
	// subtree of testDirectory2Entry.
	testCase := reconcileTestCase{
		ancestor: testDirectory2Entry,
		alpha:    testFile1Entry,
		beta:     testDirectory3Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWaySafe,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges: []*Change{
			{
				Old: testDirectory3Entry,
				New: testFile1Entry,
			},
		},
		expectedConflicts: nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaPartiallyDeletedDirectoryBetaReplacedDirectoryTwoWaySafe(t *testing.T) {
	// Set up the test case. Worth noting here is that testDirectory3Entry is a
	// subtree of testDirectory2Entry.
	testCase := reconcileTestCase{
		ancestor: testDirectory2Entry,
		alpha:    testDirectory3Entry,
		beta:     testFile1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWaySafe,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges: []*Change{
			{
				Old: testDirectory3Entry,
				New: testFile1Entry,
			},
		},
		expectedBetaChanges: nil,
		expectedConflicts:   nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaPartiallyDeletedDirectoryBetaReplacedDirectoryUnsafe(t *testing.T) {
	// Set up the test case. Worth noting here is that testDirectory3Entry is a
	// subtree of testDirectory2Entry.
	testCase := reconcileTestCase{
		ancestor: testDirectory2Entry,
		alpha:    testDirectory3Entry,
		beta:     testFile1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeTwoWayResolved,
			SynchronizationMode_SynchronizationModeOneWayReplica,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges: []*Change{
			{
				Old: testFile1Entry,
				New: testDirectory3Entry,
			},
		},
		expectedConflicts: nil,
	}

	// Run the test case.
	testCase.run(t)
}

func TestReconcileAlphaPartiallyDeletedDirectoryBetaReplacedDirectoryOneWaySafe(t *testing.T) {
	// Set up the test case. Worth noting here is that testDirectory3Entry is a
	// subtree of testDirectory2Entry.
	testCase := reconcileTestCase{
		ancestor: testDirectory2Entry,
		alpha:    testDirectory3Entry,
		beta:     testFile1Entry,
		synchronizationModes: []SynchronizationMode{
			SynchronizationMode_SynchronizationModeOneWaySafe,
		},
		expectedAncestorChanges: nil,
		expectedAlphaChanges:    nil,
		expectedBetaChanges:     nil,
		expectedConflicts: []*Conflict{
			{
				AlphaChanges: diff("", testDirectory2Entry, testDirectory3Entry),
				BetaChanges:  diff("", testDirectory2Entry, testFile1Entry),
			},
		},
	}

	// Run the test case.
	testCase.run(t)
}
