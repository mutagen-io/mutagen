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

// TestExtractNonDeletionChanges tests extractNonDeletionChanges.
func TestExtractNonDeletionChanges(t *testing.T) {
	// Define test cases.
	var tests = []struct {
		unfiltered []*Change
		expected   []*Change
	}{
		{nil, nil},
		{[]*Change{}, []*Change{}},
		{[]*Change{{New: tF1}}, []*Change{{New: tF1}}},
		{[]*Change{{New: tU}}, []*Change{{New: tU}}},
		{[]*Change{{New: tP1}}, []*Change{{New: tP1}}},
		{[]*Change{{New: tD0}, {Path: "untracked", New: tU}}, []*Change{{New: tD0}, {Path: "untracked", New: tU}}},
		{[]*Change{{Old: tF1, New: tF2}}, []*Change{{Old: tF1, New: tF2}}},
		{[]*Change{{Old: tF1}}, []*Change{}},
		{[]*Change{{Path: "changed", Old: tF1, New: tF2}, {Path: "removed", Old: tD1}}, []*Change{{Path: "changed", Old: tF1, New: tF2}}},
	}

	// Process test cases.
	for i, test := range tests {
		filtered := extractNonDeletionChanges(test.unfiltered)
		if !testingChangeListsEqual(filtered, test.expected) {
			t.Errorf("test index %d: filtered changes don't match expected: %v != %v",
				i, filtered, test.expected,
			)
		}
	}
}

// TestReconcile tests Reconcile.
func TestReconcile(t *testing.T) {
	// Define test cases.
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
			description:          "beta created file",
			modes:                twoWayModes,
			beta:                 tF1,
			expectedAlphaChanges: []*Change{{New: tF1}},
		},
		{
			description: "beta created file",
			modes:       oneWaySafeMode,
			beta:        tF1,
		},
		{
			description:         "beta created file",
			modes:               oneWayReplicaMode,
			beta:                tF1,
			expectedBetaChanges: []*Change{{Old: tF1}},
		},
		{
			description:          "beta created empty directory",
			modes:                twoWayModes,
			beta:                 tD0,
			expectedAlphaChanges: []*Change{{New: tD0}},
		},
		{
			description: "beta created empty directory",
			modes:       oneWaySafeMode,
			beta:        tD0,
		},
		{
			description:         "beta created empty directory",
			modes:               oneWayReplicaMode,
			beta:                tD0,
			expectedBetaChanges: []*Change{{Old: tD0}},
		},
		{
			description:          "beta created directory",
			modes:                twoWayModes,
			beta:                 tD1,
			expectedAlphaChanges: []*Change{{New: tD1}},
		},
		{
			description: "beta created directory",
			modes:       oneWaySafeMode,
			beta:        tD1,
		},
		{
			description:         "beta created directory",
			modes:               oneWayReplicaMode,
			beta:                tD1,
			expectedBetaChanges: []*Change{{Old: tD1}},
		},
		{
			description:          "beta created file in existing directory",
			modes:                twoWayModes,
			ancestor:             tD0,
			alpha:                tD0,
			beta:                 tD1,
			expectedAlphaChanges: []*Change{{Path: "file", New: tF1}},
		},
		{
			description: "beta created file in existing directory",
			modes:       oneWaySafeMode,
			ancestor:    tD0,
			alpha:       tD0,
			beta:        tD1,
		},
		{
			description:         "beta created file in existing directory",
			modes:               oneWayReplicaMode,
			ancestor:            tD0,
			alpha:               tD0,
			beta:                tD1,
			expectedBetaChanges: []*Change{{Path: "file", Old: tF1}},
		},
		{
			description:          "beta modified file",
			modes:                twoWayModes,
			ancestor:             tF1,
			alpha:                tF1,
			beta:                 tF2,
			expectedAlphaChanges: []*Change{{Old: tF1, New: tF2}},
		},
		{
			description: "beta modified file",
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
			description:         "beta modified file",
			modes:               oneWayReplicaMode,
			ancestor:            tF1,
			alpha:               tF1,
			beta:                tF2,
			expectedBetaChanges: []*Change{{Old: tF2, New: tF1}},
		},
		{
			description:          "beta replaced file with directory",
			modes:                twoWayModes,
			ancestor:             tF1,
			alpha:                tF1,
			beta:                 tD1,
			expectedAlphaChanges: []*Change{{Old: tF1, New: tD1}},
		},
		{
			description: "beta replaced file with directory",
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
			description:         "beta replaced file with directory",
			modes:               oneWayReplicaMode,
			ancestor:            tF1,
			alpha:               tF1,
			beta:                tD1,
			expectedBetaChanges: []*Change{{Old: tD1, New: tF1}},
		},
		{
			description:          "beta deleted file",
			modes:                twoWayModes,
			ancestor:             tF1,
			alpha:                tF1,
			expectedAlphaChanges: []*Change{{Old: tF1}},
		},
		{
			description:         "beta deleted file",
			modes:               oneWayModes,
			ancestor:            tF1,
			alpha:               tF1,
			expectedBetaChanges: []*Change{{New: tF1}},
		},
		{
			description:          "beta deleted directory",
			modes:                twoWayModes,
			ancestor:             tD1,
			alpha:                tD1,
			expectedAlphaChanges: []*Change{{Old: tD1}},
		},
		{
			description:         "beta deleted directory",
			modes:               oneWayModes,
			ancestor:            tD1,
			alpha:               tD1,
			expectedBetaChanges: []*Change{{New: tD1}},
		},

		// Test cases where both sides have modified content.
		{
			description: "both created different file",
			modes:       safeModes,
			alpha:       tF1,
			beta:        tF2,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{New: tF1}},
				BetaChanges:  []*Change{{New: tF2}},
			}},
		},
		{
			description:         "both created different file",
			modes:               resolvedModes,
			alpha:               tF1,
			beta:                tF2,
			expectedBetaChanges: []*Change{{Old: tF2, New: tF1}},
		},
		{
			description: "alpha created file beta created directory",
			modes:       safeModes,
			alpha:       tF1,
			beta:        tD1,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{New: tF1}},
				BetaChanges:  []*Change{{New: tD1}},
			}},
		},
		{
			description:         "alpha created file beta created directory",
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
			description:             "both created directory beta created file in directory",
			modes:                   twoWayModes,
			alpha:                   tD0,
			beta:                    tD1,
			expectedAncestorChanges: []*Change{{New: tD0}},
			expectedAlphaChanges:    []*Change{{Path: "file", New: tF1}},
		},
		{
			description:             "both created directory beta created file in directory",
			modes:                   oneWaySafeMode,
			alpha:                   tD0,
			beta:                    tD1,
			expectedAncestorChanges: []*Change{{New: tD0}},
		},
		{
			description:             "both created directory beta created file in directory",
			modes:                   oneWayReplicaMode,
			alpha:                   tD0,
			beta:                    tD1,
			expectedAncestorChanges: []*Change{{New: tD0}},
			expectedBetaChanges:     []*Change{{Path: "file", Old: tF1}},
		},
		{
			description:             "both created directory with different file",
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
			description:             "both created directory with different file",
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
			description:          "beta modified file alpha deleted",
			modes:                twoWayModes,
			ancestor:             tF1,
			beta:                 tF2,
			expectedAlphaChanges: []*Change{{New: tF2}},
		},
		{
			description:             "beta modified file alpha deleted",
			modes:                   oneWaySafeMode,
			ancestor:                tF1,
			beta:                    tF2,
			expectedAncestorChanges: []*Change{{}},
		},
		{
			description:         "beta modified file alpha deleted",
			modes:               oneWayReplicaMode,
			ancestor:            tF1,
			beta:                tF2,
			expectedBetaChanges: []*Change{{Old: tF2}},
		},
		{
			description:         "alpha deleted all of directory, beta deleted part of directory",
			modes:               allModes,
			ancestor:            tD1,
			beta:                tD0,
			expectedBetaChanges: []*Change{{Old: tD0}},
		},
		{
			description:          "alpha deleted part of directory, beta deleted all of directory",
			modes:                twoWayModes,
			ancestor:             tD1,
			alpha:                tD0,
			expectedAlphaChanges: []*Change{{Old: tD0}},
		},
		{
			description:         "alpha deleted part of directory, beta deleted all of directory",
			modes:               oneWayModes,
			ancestor:            tD1,
			alpha:               tD0,
			expectedBetaChanges: []*Change{{New: tD0}},
		},

		// Test cases with a combination of synchronizable and unsynchronizable
		// content, including cases with modified content.
		{
			description: "alpha created untracked, beta created file",
			modes:       twoWayModes,
			alpha:       tU,
			beta:        tF1,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{New: tU}},
				BetaChanges:  []*Change{{New: tF1}},
			}},
		},
		{
			description: "alpha created untracked, beta created file",
			modes:       oneWaySafeMode,
			alpha:       tU,
			beta:        tF1,
		},
		{
			description:         "alpha created untracked, beta created file",
			modes:               oneWayReplicaMode,
			alpha:               tU,
			beta:                tF1,
			expectedBetaChanges: []*Change{{Old: tF1}},
		},
		{
			description: "alpha created problematic in existing directory",
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
			description: "alpha created file, beta created directory with untracked",
			modes:       safeModes,
			alpha:       tF1,
			beta:        tDU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{New: tF1}},
				BetaChanges:  []*Change{{New: tD0}},
			}},
		},
		{
			description: "alpha created file, beta created directory with untracked",
			modes:       resolvedModes,
			alpha:       tF1,
			beta:        tDU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{New: tF1}},
				BetaChanges:  []*Change{{Path: "untracked", New: tU}},
			}},
		},
		{
			description: "beta created problematic in existing directory",
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
			description:          "beta created directory with problematic",
			modes:                twoWayModes,
			beta:                 tDP1,
			expectedAlphaChanges: []*Change{{New: tD0}},
		},
		{
			description: "beta created directory with problematic",
			modes:       oneWaySafeMode,
			beta:        tDP1,
		},
		{
			description: "beta created directory with problematic",
			modes:       oneWayReplicaMode,
			beta:        tDP1,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{}},
				BetaChanges:  []*Change{{Path: "problematic", New: tP1}},
			}},
		},
		{
			description:          "beta created directory with untracked",
			modes:                twoWayModes,
			beta:                 tDU,
			expectedAlphaChanges: []*Change{{New: tD0}},
		},
		{
			description: "beta created directory with untracked",
			modes:       oneWaySafeMode,
			beta:        tDU,
		},
		{
			description: "beta created directory with untracked",
			modes:       oneWayReplicaMode,
			beta:        tDU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{}},
				BetaChanges:  []*Change{{Path: "untracked", New: tU}},
			}},
		},
		{
			description:          "beta replaced file with untracked",
			modes:                twoWayModes,
			ancestor:             tF1,
			alpha:                tF1,
			beta:                 tU,
			expectedAlphaChanges: []*Change{{Old: tF1}},
		},
		{
			description: "beta replaced file with untracked",
			modes:       oneWayModes,
			ancestor:    tF1,
			alpha:       tF1,
			beta:        tU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Old: tF1, New: tF1}},
				BetaChanges:  []*Change{{New: tU}},
			}},
		},
		{
			description:          "beta replaced file with directory containing untracked",
			modes:                twoWayModes,
			ancestor:             tF1,
			alpha:                tF1,
			beta:                 tDU,
			expectedAlphaChanges: []*Change{{Old: tF1, New: tD0}},
		},
		{
			description: "beta replaced file with directory containing untracked",
			modes:       oneWaySafeMode,
			ancestor:    tF1,
			alpha:       tF1,
			beta:        tDU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Old: tF1, New: tF1}},
				BetaChanges:  []*Change{{Old: tF1, New: tD0}},
			}},
		},
		{
			description: "beta replaced file with directory containing untracked",
			modes:       oneWayReplicaMode,
			ancestor:    tF1,
			alpha:       tF1,
			beta:        tDU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Old: tF1, New: tF1}},
				BetaChanges:  []*Change{{Path: "untracked", New: tU}},
			}},
		},
		{
			description: "alpha deleted all of directory, beta deleted part of directory and created unsynchronizable content",
			modes:       allModes,
			ancestor:    tD1,
			beta:        tDU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Old: tD1}},
				BetaChanges:  []*Change{{Path: "untracked", New: tU}},
			}},
		},
		{
			description: "alpha deleted part of directory and created unsynchronizable content, beta deleted all of directory",
			modes:       twoWayModes,
			ancestor:    tD1,
			alpha:       tDU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Path: "untracked", New: tU}},
				BetaChanges:  []*Change{{Old: tD1}},
			}},
		},
		{
			description:         "alpha deleted part of directory and created unsynchronizable content, beta deleted all of directory",
			modes:               oneWayModes,
			ancestor:            tD1,
			alpha:               tDU,
			expectedBetaChanges: []*Change{{New: tD0}},
		},
		{
			description:         "alpha replaced directory with unsynchronizable content, beta deleted part of directory",
			modes:               allModes,
			ancestor:            tD1,
			alpha:               tU,
			beta:                tD0,
			expectedBetaChanges: []*Change{{Old: tD0}},
		},
		{
			description:          "alpha deleted part of directory, beta replaced directory with unsynchronizable content",
			modes:                twoWayModes,
			ancestor:             tD1,
			alpha:                tD0,
			beta:                 tU,
			expectedAlphaChanges: []*Change{{Old: tD0}},
		},
		{
			description: "alpha deleted part of directory, beta replaced directory with unsynchronizable content",
			modes:       oneWayModes,
			ancestor:    tD1,
			alpha:       tD0,
			beta:        tU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Old: tD1, New: tD0}},
				BetaChanges:  []*Change{{New: tU}},
			}},
		},
		{
			description: "alpha deleted part of directory and created untracked content, beta replaced directory with file",
			modes:       twoWayModes,
			ancestor:    tD1,
			alpha:       tDU,
			beta:        tF2,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Path: "untracked", New: tU}},
				BetaChanges:  []*Change{{Old: tD1, New: tF2}},
			}},
		},
		{
			description: "alpha deleted part of directory and created untracked content, beta replaced directory with file",
			modes:       oneWaySafeMode,
			ancestor:    tD1,
			alpha:       tDU,
			beta:        tF2,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Old: tD1, New: tDU}},
				BetaChanges:  []*Change{{Old: tD1, New: tF2}},
			}},
		},
		{
			description:         "alpha deleted part of directory and created untracked content, beta replaced directory with file",
			modes:               oneWayReplicaMode,
			ancestor:            tD1,
			alpha:               tDU,
			beta:                tF2,
			expectedBetaChanges: []*Change{{Old: tF2, New: tD0}},
		},
		{
			description: "beta deleted part of directory and created untracked content, alpha replaced directory with file",
			modes:       allModes,
			ancestor:    tD1,
			alpha:       tF2,
			beta:        tDU,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Old: tD1, New: tF2}},
				BetaChanges:  []*Change{{Path: "untracked", New: tU}},
			}},
		},
		{
			description: "alpha deleted part of directory and created problematic content, beta replaced directory with file",
			modes:       twoWayModes,
			ancestor:    tD1,
			alpha:       tDP1,
			beta:        tF2,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Path: "problematic", New: tP1}},
				BetaChanges:  []*Change{{Old: tD1, New: tF2}},
			}},
		},
		{
			description: "alpha deleted part of directory and created problematic content, beta replaced directory with file",
			modes:       oneWaySafeMode,
			ancestor:    tD1,
			alpha:       tDP1,
			beta:        tF2,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Old: tD1, New: tDP1}},
				BetaChanges:  []*Change{{Old: tD1, New: tF2}},
			}},
		},
		{
			description:         "alpha deleted part of directory and created problematic content, beta replaced directory with file",
			modes:               oneWayReplicaMode,
			ancestor:            tD1,
			alpha:               tDP1,
			beta:                tF2,
			expectedBetaChanges: []*Change{{Old: tF2, New: tD0}},
		},
		{
			description: "beta deleted part of directory and created problematic content, alpha replaced directory with file",
			modes:       allModes,
			ancestor:    tD1,
			alpha:       tF2,
			beta:        tDP1,
			expectedConflicts: []*Conflict{{
				AlphaChanges: []*Change{{Old: tD1, New: tF2}},
				BetaChanges:  []*Change{{Path: "problematic", New: tP1}},
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
				t.Errorf("%s (%s): ancestor changes do not match expected: %v != %v",
					test.description, mode.Description(), ancestorChanges, test.expectedAncestorChanges,
				)
			}

			// Verify the alpha changes.
			if !testingChangeListsEqual(alphaChanges, test.expectedAlphaChanges) {
				t.Errorf("%s (%s): alpha changes do not match expected: %v != %v",
					test.description, mode.Description(), alphaChanges, test.expectedAlphaChanges,
				)
			}

			// Verify the beta changes.
			if !testingChangeListsEqual(betaChanges, test.expectedBetaChanges) {
				t.Errorf("%s (%s): beta changes do not match expected: %v != %v",
					test.description, mode.Description(), betaChanges, test.expectedBetaChanges,
				)
			}

			// Verify the conflicts.
			if !testingConflictListsEqual(conflicts, test.expectedConflicts) {
				t.Errorf("%s (%s): conflicts do not match expected: %v != %v",
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
