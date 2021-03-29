package core

import (
	"testing"
)

// allModes is shorthand for all synchronization modes.
var allModes = []SynchronizationMode{
	SynchronizationMode_SynchronizationModeTwoWaySafe,
	SynchronizationMode_SynchronizationModeTwoWayResolved,
	SynchronizationMode_SynchronizationModeOneWaySafe,
	SynchronizationMode_SynchronizationModeOneWayReplica,
}

// twoWayModes is shorthand for all bidirectional synchronization modes.
var twoWayModes = []SynchronizationMode{
	SynchronizationMode_SynchronizationModeTwoWaySafe,
	SynchronizationMode_SynchronizationModeTwoWayResolved,
}

// oneWayModes is shorthand for all unidirectional synchronization modes.
var oneWayModes = []SynchronizationMode{
	SynchronizationMode_SynchronizationModeOneWaySafe,
	SynchronizationMode_SynchronizationModeOneWayReplica,
}

// safeModes is shorthand for all safe synchronization modes.
var safeModes = []SynchronizationMode{
	SynchronizationMode_SynchronizationModeTwoWaySafe,
	SynchronizationMode_SynchronizationModeOneWaySafe,
}

// resolvedModes is shorthand for all resolved synchronization modes.
var resolvedModes = []SynchronizationMode{
	SynchronizationMode_SynchronizationModeTwoWayResolved,
	SynchronizationMode_SynchronizationModeOneWayReplica,
}

// twoWaySafeMode is shorthand for only the two-way-safe mode.
var twoWaySafeMode = []SynchronizationMode{
	SynchronizationMode_SynchronizationModeTwoWaySafe,
}

// twoWayResolvedMode is shorthand for only the two-way-resolved mode.
var twoWayResolvedMode = []SynchronizationMode{
	SynchronizationMode_SynchronizationModeTwoWayResolved,
}

// oneWaySafeMode is shorthand for only the one-way-safe mode.
var oneWaySafeMode = []SynchronizationMode{
	SynchronizationMode_SynchronizationModeOneWaySafe,
}

// oneWayReplicaMode is shorthand for only the one-way-replica mode.
var oneWayReplicaMode = []SynchronizationMode{
	SynchronizationMode_SynchronizationModeOneWayReplica,
}

// TestNonDeletionChangesOnly tests nonDeletionChangesOnly.
func TestNonDeletionChangesOnly(t *testing.T) {
	// Set up test cases.
	var testCases = []struct {
		unfiltered []*Change
		expected   []*Change
	}{
		{nil, nil},
		{[]*Change{}, []*Change{}},
		{[]*Change{{New: tF1}}, []*Change{{New: tF1}}},
		{[]*Change{{Old: tF1, New: tF2}}, []*Change{{Old: tF1, New: tF2}}},
		{[]*Change{{Old: tF1}}, []*Change{}},
		{[]*Change{{Path: "changed", Old: tF1, New: tF2}, {Path: "removed", Old: tD1}}, []*Change{{Path: "changed", Old: tF1, New: tF2}}},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if filtered := nonDeletionChangesOnly(testCase.unfiltered); !testingChangeListsEqual(filtered, testCase.expected) {
			t.Errorf("filtered changes don't match expected: %v != %v", filtered, testCase.expected)
		}
	}
}

// TestReconcile tests Reconcile.
func TestReconcile(t *testing.T) {
	// Set up test cases.
	var tests = []struct {
		// description is a human readable description of the test case.
		description string
		// modes are the synchronization modes for which the test case should apply.
		modes []SynchronizationMode
		// ancestor is the root ancestor entry.
		ancestor *Entry
		// alpha is the root alpha entry.
		alpha *Entry
		// beta is the root beta entry.
		beta *Entry
		// expectedAncestorChanges are the expected ancestor changes.
		expectedAncestorChanges []*Change
		// expectedAlphaChanges are the expected alpha changes.
		expectedAlphaChanges []*Change
		// expectedBetaChanges are the expected beta changes.
		expectedBetaChanges []*Change
		// expectedConflicts are the expected conflicts.
		expectedConflicts []*Conflict
	}{
		// Test cases where alpha and beta (and potentially the ancestor) agree.
		{
			description: "all nil",
			modes:       allModes,
		},
		{
			description: "all same file",
			modes:       allModes,
			ancestor:    tF1,
			alpha:       tF1,
			beta:        tF1,
		},
		{
			description: "all same relative symbolic link",
			modes:       allModes,
			ancestor:    tSR,
			alpha:       tSR,
			beta:        tSR,
		},
		{
			description: "all same absolute symbolic link",
			modes:       allModes,
			ancestor:    tSA,
			alpha:       tSA,
			beta:        tSA,
		},
		{
			description: "all same empty directory",
			modes:       allModes,
			ancestor:    tD0,
			alpha:       tD0,
			beta:        tD0,
		},
		{
			description: "all same directory",
			modes:       allModes,
			ancestor:    tD1,
			alpha:       tD1,
			beta:        tD1,
		},
		{
			description:             "both created same file",
			modes:                   allModes,
			alpha:                   tF1,
			beta:                    tF1,
			expectedAncestorChanges: []*Change{{New: tF1}},
		},
		{
			description:             "both created same relative symbolic link",
			modes:                   allModes,
			alpha:                   tSR,
			beta:                    tSR,
			expectedAncestorChanges: []*Change{{New: tSR}},
		},
		{
			description:             "both created same absolute symbolic link",
			modes:                   allModes,
			alpha:                   tSA,
			beta:                    tSA,
			expectedAncestorChanges: []*Change{{New: tSA}},
		},
		{
			description:             "both created same empty directory",
			modes:                   allModes,
			alpha:                   tD0,
			beta:                    tD0,
			expectedAncestorChanges: []*Change{{New: tD0}},
		},
		{
			description: "both created same directory",
			modes:       allModes,
			alpha:       tD1,
			beta:        tD1,
			expectedAncestorChanges: []*Change{
				{New: tD0},
				{Path: "file", New: tF1},
			},
		},
		{
			description:             "both created same file in directory",
			modes:                   allModes,
			ancestor:                tD0,
			alpha:                   tD1,
			beta:                    tD1,
			expectedAncestorChanges: []*Change{{Path: "file", New: tF1}},
		},
		{
			description:             "both created same relative symbolic link in directory",
			modes:                   allModes,
			ancestor:                tD1,
			alpha:                   tDSR,
			beta:                    tDSR,
			expectedAncestorChanges: []*Change{{Path: "symlink", New: tSR}},
		},
		{
			description:             "both created same absolute symbolic link in directory",
			modes:                   allModes,
			ancestor:                tD0,
			alpha:                   tDSA,
			beta:                    tDSA,
			expectedAncestorChanges: []*Change{{Path: "symlink", New: tSA}},
		},
		{
			description:             "both deleted file",
			modes:                   allModes,
			ancestor:                tF1,
			expectedAncestorChanges: []*Change{{}},
		},
		{
			description:             "both deleted file in directory",
			modes:                   allModes,
			ancestor:                tD1,
			alpha:                   tD0,
			beta:                    tD0,
			expectedAncestorChanges: []*Change{{Path: "file"}},
		},
		{
			description:             "both deleted relative symbolic link in directory",
			modes:                   allModes,
			ancestor:                tDSR,
			alpha:                   tD1,
			beta:                    tD1,
			expectedAncestorChanges: []*Change{{Path: "symlink"}},
		},
		{
			description:             "both deleted absolute symbolic link in directory",
			modes:                   allModes,
			ancestor:                tDSA,
			alpha:                   tD0,
			beta:                    tD0,
			expectedAncestorChanges: []*Change{{Path: "symlink"}},
		},
		{
			description:             "both deleted empty directory",
			modes:                   allModes,
			ancestor:                tD0,
			expectedAncestorChanges: []*Change{{}},
		},
		{
			description:             "both deleted directory",
			modes:                   allModes,
			ancestor:                tD1,
			expectedAncestorChanges: []*Change{{}},
		},

		// Test cases where both sides are either nil or untracked.
		{
			description: "beta created untracked",
			modes:       allModes,
			beta:        tU,
		},
		{
			description: "alpha created untracked",
			modes:       allModes,
			alpha:       tU,
		},
		{
			description: "both created untracked",
			modes:       allModes,
			alpha:       tU,
			beta:        tU,
		},
		{
			description:             "alpha deleted file and beta created untracked",
			modes:                   allModes,
			ancestor:                tF1,
			beta:                    tU,
			expectedAncestorChanges: []*Change{{}},
		},
		{
			description:             "alpha created untracked and beta deleted file",
			modes:                   allModes,
			ancestor:                tF1,
			alpha:                   tU,
			expectedAncestorChanges: []*Change{{}},
		},
		{
			description:             "both deleted directory and created untracked",
			modes:                   allModes,
			ancestor:                tD1,
			alpha:                   tU,
			beta:                    tU,
			expectedAncestorChanges: []*Change{{}},
		},

		// Test cases where one or both sides is problematic.
		{
			description: "alpha problematic others nil",
			modes:       allModes,
			alpha:       tP1,
		},
		{
			description: "alpha problematic ancestor file",
			modes:       allModes,
			ancestor:    tF1,
			alpha:       tP1,
		},
		{
			description: "alpha problematic ancestor empty directory",
			modes:       allModes,
			ancestor:    tD0,
			alpha:       tP1,
		},
		{
			description: "alpha problematic others file",
			modes:       allModes,
			ancestor:    tF1,
			alpha:       tP1,
			beta:        tF1,
		},
		{
			description: "beta problematic others nil",
			modes:       allModes,
			beta:        tP1,
		},
		{
			description: "beta problematic ancestor file",
			modes:       allModes,
			ancestor:    tF1,
			beta:        tP1,
		},
		{
			description: "beta problematic others file",
			modes:       allModes,
			ancestor:    tF1,
			alpha:       tF1,
			beta:        tP1,
		},
		{
			description: "beta problematic ancestor empty directory",
			modes:       allModes,
			ancestor:    tD0,
			beta:        tP1,
		},
		{
			description: "both problematic ancestor nil",
			modes:       allModes,
			alpha:       tP1,
			beta:        tP1,
		},
		{
			description: "both problematic ancestor file",
			modes:       allModes,
			ancestor:    tF1,
			alpha:       tP1,
			beta:        tP1,
		},
		{
			description: "both problematic ancestor directory",
			modes:       allModes,
			ancestor:    tD1,
			alpha:       tP1,
			beta:        tP1,
		},

		// Test cases where only alpha has modified content.
		{
			description:         "alpha created file",
			modes:               allModes,
			alpha:               tF1,
			expectedBetaChanges: []*Change{{New: tF1}},
		},
		{
			description:         "alpha created empty directory",
			modes:               allModes,
			alpha:               tD0,
			expectedBetaChanges: []*Change{{New: tD0}},
		},
		{
			description:         "alpha created directory",
			modes:               allModes,
			alpha:               tD1,
			expectedBetaChanges: []*Change{{New: tD1}},
		},
		{
			description:         "alpha created file in existing directory",
			modes:               allModes,
			ancestor:            tD0,
			alpha:               tD1,
			beta:                tD0,
			expectedBetaChanges: []*Change{{Path: "file", New: tF1}},
		},
		{
			description:         "alpha modified file",
			modes:               allModes,
			ancestor:            tF1,
			alpha:               tF2,
			beta:                tF1,
			expectedBetaChanges: []*Change{{Old: tF1, New: tF2}},
		},
		{
			description:         "alpha replaced file with directory",
			modes:               allModes,
			ancestor:            tF1,
			alpha:               tD1,
			beta:                tF1,
			expectedBetaChanges: []*Change{{Old: tF1, New: tD1}},
		},
		{
			description:         "alpha deleted file",
			modes:               allModes,
			ancestor:            tF1,
			beta:                tF1,
			expectedBetaChanges: []*Change{{Old: tF1}},
		},
		{
			description:         "alpha deleted directory",
			modes:               allModes,
			ancestor:            tD1,
			beta:                tD1,
			expectedBetaChanges: []*Change{{Old: tD1}},
		},

		// Test cases where only beta has modified content.
		{
			description:          "beta created file (bidirectional)",
			modes:                twoWayModes,
			beta:                 tF1,
			expectedAlphaChanges: []*Change{{New: tF1}},
		},
		{
			description: "beta created file (one-way-safe)",
			modes:       oneWaySafeMode,
			beta:        tF1,
		},
		{
			description:         "beta created file (one-way-replica)",
			modes:               oneWayReplicaMode,
			beta:                tF1,
			expectedBetaChanges: []*Change{{Old: tF1}},
		},
		{
			description:          "beta created empty directory (bidirectional)",
			modes:                twoWayModes,
			beta:                 tD0,
			expectedAlphaChanges: []*Change{{New: tD0}},
		},
		{
			description: "beta created empty directory (one-way-safe)",
			modes:       oneWaySafeMode,
			beta:        tD0,
		},
		{
			description:         "beta created empty directory (one-way-replica)",
			modes:               oneWayReplicaMode,
			beta:                tD0,
			expectedBetaChanges: []*Change{{Old: tD0}},
		},
		{
			description:          "beta created directory (bidirectional)",
			modes:                twoWayModes,
			beta:                 tD1,
			expectedAlphaChanges: []*Change{{New: tD1}},
		},
		{
			description: "beta created directory (one-way-safe)",
			modes:       oneWaySafeMode,
			beta:        tD1,
		},
		{
			description:         "beta created directory (one-way-replica)",
			modes:               oneWayReplicaMode,
			beta:                tD1,
			expectedBetaChanges: []*Change{{Old: tD1}},
		},
		{
			description:          "beta created file in existing directory (bidirectional)",
			modes:                twoWayModes,
			ancestor:             tD0,
			alpha:                tD0,
			beta:                 tD1,
			expectedAlphaChanges: []*Change{{Path: "file", New: tF1}},
		},
		{
			description: "beta created file in existing directory (one-way-safe)",
			modes:       oneWaySafeMode,
			ancestor:    tD0,
			alpha:       tD0,
			beta:        tD1,
		},
		{
			description:         "beta created file in existing directory (one-way-replica)",
			modes:               oneWayReplicaMode,
			ancestor:            tD0,
			alpha:               tD0,
			beta:                tD1,
			expectedBetaChanges: []*Change{{Path: "file", Old: tF1}},
		},
		{
			description:          "beta modified file (bidirectional)",
			modes:                twoWayModes,
			ancestor:             tF1,
			alpha:                tF1,
			beta:                 tF2,
			expectedAlphaChanges: []*Change{{Old: tF1, New: tF2}},
		},
		{
			description: "beta modified file (one-way-safe)",
			modes:       oneWaySafeMode,
			ancestor:    tF1,
			alpha:       tF1,
			beta:        tF2,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Old: tF1, New: tF1}},
				BetaChanges:  []*Change{{Old: tF1, New: tF2}},
			}},
		},
		{
			description:         "beta modified file (one-way-replica)",
			modes:               oneWayReplicaMode,
			ancestor:            tF1,
			alpha:               tF1,
			beta:                tF2,
			expectedBetaChanges: []*Change{{Old: tF2, New: tF1}},
		},
		{
			description:          "beta replaced file with directory (bidirectional)",
			modes:                twoWayModes,
			ancestor:             tF1,
			alpha:                tF1,
			beta:                 tD1,
			expectedAlphaChanges: []*Change{{Old: tF1, New: tD1}},
		},
		{
			description: "beta replaced file with directory (one-way-safe)",
			modes:       oneWaySafeMode,
			ancestor:    tF1,
			alpha:       tF1,
			beta:        tD1,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Old: tF1, New: tF1}},
				BetaChanges:  []*Change{{Old: tF1, New: tD1}},
			}},
		},
		{
			description:         "beta replaced file with directory (one-way-replica)",
			modes:               oneWayReplicaMode,
			ancestor:            tF1,
			alpha:               tF1,
			beta:                tD1,
			expectedBetaChanges: []*Change{{Old: tD1, New: tF1}},
		},
		{
			description:          "beta deleted file (bidirectional)",
			modes:                twoWayModes,
			ancestor:             tF1,
			alpha:                tF1,
			expectedAlphaChanges: []*Change{{Old: tF1}},
		},
		{
			description:         "beta deleted file (unidirectional)",
			modes:               oneWayModes,
			ancestor:            tF1,
			alpha:               tF1,
			expectedBetaChanges: []*Change{{New: tF1}},
		},
		{
			description:          "beta deleted directory (bidirectional)",
			modes:                twoWayModes,
			ancestor:             tD1,
			alpha:                tD1,
			expectedAlphaChanges: []*Change{{Old: tD1}},
		},
		{
			description:         "beta deleted directory (unidirectional)",
			modes:               oneWayModes,
			ancestor:            tD1,
			alpha:               tD1,
			expectedBetaChanges: []*Change{{New: tD1}},
		},

		// Test cases where both sides have modified content.
		{
			description: "both created different file (safe)",
			modes:       safeModes,
			alpha:       tF1,
			beta:        tF2,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{New: tF1}},
				BetaChanges:  []*Change{{New: tF2}},
			}},
		},
		{
			description:         "both created different file (resolved)",
			modes:               resolvedModes,
			alpha:               tF1,
			beta:                tF2,
			expectedBetaChanges: []*Change{{Old: tF2, New: tF1}},
		},
		{
			description: "alpha created file beta created directory (safe)",
			modes:       safeModes,
			alpha:       tF1,
			beta:        tD1,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{New: tF1}},
				BetaChanges:  []*Change{{New: tD1}},
			}},
		},
		{
			description:         "alpha created file beta created directory (resolved)",
			modes:               resolvedModes,
			alpha:               tF1,
			beta:                tD1,
			expectedBetaChanges: []*Change{{Old: tD1, New: tF1}},
		},
		{
			description:             "both created directory alpha created file in directory",
			modes:                   allModes,
			alpha:                   tD1,
			beta:                    tD0,
			expectedAncestorChanges: []*Change{{New: tD0}},
			expectedBetaChanges:     []*Change{{Path: "file", New: tF1}},
		},
		{
			description:             "both created directory beta created file in directory (bidirectional)",
			modes:                   twoWayModes,
			alpha:                   tD0,
			beta:                    tD1,
			expectedAncestorChanges: []*Change{{New: tD0}},
			expectedAlphaChanges:    []*Change{{Path: "file", New: tF1}},
		},
		{
			description:             "both created directory beta created file in directory (one-way-safe)",
			modes:                   oneWaySafeMode,
			alpha:                   tD0,
			beta:                    tD1,
			expectedAncestorChanges: []*Change{{New: tD0}},
		},
		{
			description:             "both created directory beta created file in directory (one-way-replica)",
			modes:                   oneWayReplicaMode,
			alpha:                   tD0,
			beta:                    tD1,
			expectedAncestorChanges: []*Change{{New: tD0}},
			expectedBetaChanges:     []*Change{{Path: "file", Old: tF1}},
		},
		{
			description:             "both created directory with different file (safe)",
			modes:                   safeModes,
			alpha:                   tD1,
			beta:                    tD2,
			expectedAncestorChanges: []*Change{{New: tD0}},
			expectedConflicts: []*Conflict{{
				Root:         "file",
				AlphaChanges: []*Change{{Path: "file", New: tF1}},
				BetaChanges:  []*Change{{Path: "file", New: tF2}},
			}},
		},
		{
			description:             "both created directory with different file (resolved)",
			modes:                   resolvedModes,
			alpha:                   tD1,
			beta:                    tD2,
			expectedAncestorChanges: []*Change{{New: tD0}},
			expectedBetaChanges:     []*Change{{Path: "file", Old: tF2, New: tF1}},
		},
		{
			description:         "alpha modified file beta deleted",
			modes:               allModes,
			ancestor:            tF1,
			alpha:               tF2,
			expectedBetaChanges: []*Change{{New: tF2}},
		},
		{
			description:          "beta modified file alpha deleted (bidirectional)",
			modes:                twoWayModes,
			ancestor:             tF1,
			beta:                 tF2,
			expectedAlphaChanges: []*Change{{New: tF2}},
		},
		{
			description:             "beta modified file alpha deleted (one-way-safe)",
			modes:                   oneWaySafeMode,
			ancestor:                tF1,
			beta:                    tF2,
			expectedAncestorChanges: []*Change{{}},
		},
		{
			description:         "beta modified file alpha deleted (one-way-replica)",
			modes:               oneWayReplicaMode,
			ancestor:            tF1,
			beta:                tF2,
			expectedBetaChanges: []*Change{{Old: tF2}},
		},
		{
			description:         "alpha deleted all, beta deleted part of directory",
			modes:               allModes,
			ancestor:            tD1,
			beta:                tD0,
			expectedBetaChanges: []*Change{{Old: tD0}},
		},
		{
			description:          "alpha deleted part, beta deleted all of directory (bidirectional)",
			modes:                twoWayModes,
			ancestor:             tD1,
			alpha:                tD0,
			expectedAlphaChanges: []*Change{{Old: tD0}},
		},
		{
			description:         "alpha deleted part, beta deleted all of directory (unidirectional)",
			modes:               oneWayModes,
			ancestor:            tD1,
			alpha:               tD0,
			expectedBetaChanges: []*Change{{New: tD0}},
		},

		// Test cases with a combination of synchronizable and unsynchronizable
		// content, including cases with modified content.
		{
			description: "alpha created untracked, beta created file (safe)",
			modes:       safeModes,
			alpha:       tU,
			beta:        tF1,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{New: tU}},
				BetaChanges:  []*Change{{New: tF1}},
			}},
		},
		{
			description:         "alpha created untracked, beta created file (resolved)",
			modes:               resolvedModes,
			alpha:               tU,
			beta:                tF1,
			expectedBetaChanges: []*Change{{Old: tF1}},
		},
		{
			description: "alpha problematic in existing directory",
			modes:       allModes,
			ancestor:    tD0,
			alpha:       tDP1,
			beta:        tD0,
		},
		{
			description: "alpha created untracked in existing directory",
			modes:       allModes,
			ancestor:    tD0,
			alpha:       tDU,
			beta:        tD0,
		},
		{
			description:         "alpha created directory with problematic",
			modes:               allModes,
			alpha:               tDP1,
			expectedBetaChanges: []*Change{{New: tD0}},
		},
		{
			description:         "alpha created directory with untracked",
			modes:               allModes,
			alpha:               tDU,
			expectedBetaChanges: []*Change{{New: tD0}},
		},
		{
			description: "alpha created file, beta created untracked",
			modes:       allModes,
			alpha:       tF1,
			beta:        tU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{New: tF1}},
				BetaChanges:  []*Change{{New: tU}},
			}},
		},
		{
			description: "alpha created file, beta directory with untracked (safe)",
			modes:       safeModes,
			alpha:       tF1,
			beta:        tDU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{New: tF1}},
				BetaChanges:  []*Change{{New: tDU}},
			}},
		},
		{
			description: "alpha created file, beta directory with untracked (resolved)",
			modes:       resolvedModes,
			alpha:       tF1,
			beta:        tDU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{New: tF1}},
				BetaChanges:  []*Change{{Path: "untracked", New: tU}},
			}},
		},

		{
			description: "alpha created file, beta created untracked",
			modes:       allModes,
			alpha:       tF1,
			beta:        tU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{New: tF1}},
				BetaChanges:  []*Change{{New: tU}},
			}},
		},
		{
			description: "beta problematic in existing directory",
			modes:       allModes,
			ancestor:    tD0,
			alpha:       tD0,
			beta:        tDP1,
		},
		{
			description: "beta created untracked in existing directory",
			modes:       allModes,
			ancestor:    tD0,
			alpha:       tDU,
			beta:        tD0,
		},
		{
			description:          "beta created directory with problematic (bidirectional)",
			modes:                twoWayModes,
			beta:                 tDP1,
			expectedAlphaChanges: []*Change{{New: tD0}},
		},
		{
			description: "beta created directory with problematic (one-way-safe)",
			modes:       oneWaySafeMode,
			beta:        tDP1,
		},
		{
			description: "beta created directory with problematic (one-way-replica)",
			modes:       oneWayReplicaMode,
			beta:        tDP1,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{}},
				BetaChanges:  []*Change{{Path: "problematic", New: tP1}},
			}},
		},
		{
			description:          "beta created directory with untracked (bidirectional)",
			modes:                twoWayModes,
			beta:                 tDU,
			expectedAlphaChanges: []*Change{{New: tD0}},
		},
		{
			description: "beta created directory with untracked (one-way-safe)",
			modes:       oneWaySafeMode,
			beta:        tDU,
		},
		{
			description: "beta created directory with untracked (one-way-replica)",
			modes:       oneWayReplicaMode,
			beta:        tDU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{}},
				BetaChanges:  []*Change{{Path: "untracked", New: tU}},
			}},
		},
		{
			description: "alpha created untracked, beta created file (safe)",
			modes:       safeModes,
			alpha:       tU,
			beta:        tF1,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{New: tU}},
				BetaChanges:  []*Change{{New: tF1}},
			}},
		},
		{
			description:         "alpha created untracked, beta created file (resolved)",
			modes:               resolvedModes,
			alpha:               tU,
			beta:                tF1,
			expectedBetaChanges: []*Change{{Old: tF1}},
		},
		{
			description:          "beta replaced file with untracked (bidirectional)",
			modes:                twoWayModes,
			ancestor:             tF1,
			alpha:                tF1,
			beta:                 tU,
			expectedAlphaChanges: []*Change{{Old: tF1}},
		},
		{
			description: "beta replaced file with untracked (one-way-safe)",
			modes:       oneWaySafeMode,
			ancestor:    tF1,
			alpha:       tF1,
			beta:        tU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Old: tF1, New: tF1}},
				BetaChanges:  []*Change{{Old: tF1, New: tU}},
			}},
		},
		{
			description: "beta replaced file with untracked (one-way-replica)",
			modes:       oneWayReplicaMode,
			ancestor:    tF1,
			alpha:       tF1,
			beta:        tU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Old: tF1, New: tF1}},
				BetaChanges:  []*Change{{New: tU}},
			}},
		},
		{
			description:          "beta replaced file with directory containing untracked (bidirectional)",
			modes:                twoWayModes,
			ancestor:             tF1,
			alpha:                tF1,
			beta:                 tDU,
			expectedAlphaChanges: []*Change{{Old: tF1, New: tD0}},
		},
		{
			description: "beta replaced file with directory containing untracked (one-way-safe)",
			modes:       oneWaySafeMode,
			ancestor:    tF1,
			alpha:       tF1,
			beta:        tDU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Old: tF1, New: tF1}},
				BetaChanges:  []*Change{{Old: tF1, New: tDU}},
			}},
		},
		{
			description: "beta replaced file with directory containing untracked (one-way-replica)",
			modes:       oneWayReplicaMode,
			ancestor:    tF1,
			alpha:       tF1,
			beta:        tDU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Old: tF1, New: tF1}},
				BetaChanges:  []*Change{{Path: "untracked", New: tU}},
			}},
		},
	}

	// Process test cases.
	for _, test := range tests {
		// Test each mode.
		for _, mode := range test.modes {
			// Perform reconciliation.
			ancestorChanges, alphaChanges, betaChanges, conflicts := Reconcile(
				test.ancestor, test.alpha, test.beta, mode,
			)

			// Verify the ancestor changes.
			if !testingChangeListsEqual(ancestorChanges, test.expectedAncestorChanges) {
				t.Errorf("%s: ancestor changes do not match expected in %s mode: %v != %v",
					test.description, mode.Description(), ancestorChanges, test.expectedAncestorChanges,
				)
			}

			// Verify the alpha changes.
			if !testingChangeListsEqual(alphaChanges, test.expectedAlphaChanges) {
				t.Errorf("%s: alpha changes do not match expected in %s mode: %v != %v",
					test.description, mode.Description(), alphaChanges, test.expectedAlphaChanges,
				)
			}

			// Verify the beta changes.
			if !testingChangeListsEqual(betaChanges, test.expectedBetaChanges) {
				t.Errorf("%s: beta changes do not match expected in %s mode: %v != %v",
					test.description, mode.Description(), betaChanges, test.expectedBetaChanges,
				)
			}

			// Verify the conflicts.
			if !testingConflictListsEqual(conflicts, test.expectedConflicts) {
				t.Errorf("%s: conflicts do not match expected in %s mode: %v != %v",
					test.description, mode.Description(), conflicts, test.expectedConflicts,
				)
			}
		}
	}
}

// TestReconcilePanicWithInvalidSynchronizationMode tests that Reconcile panics
// when provided with disagreeing contents and an invalid synchronization mode.
func TestReconcilePanicWithInvalidSynchronizationMode(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Reconcile did not panic with invalid synchronization mode")
		}
	}()
	Reconcile(nil, tF1, nil, SynchronizationMode(-1))
}
