package core

// testingChangeListsEqual determines whether or not two lists of changes are
// equivalent. The change lists do not need to be in the same order, but they do
// need to be structurally equivalent (i.e. not composed differently).
func testingChangeListsEqual(actualChanges, expectedChanges []*Change) bool {
	// Verify that the number of changes is the same in each case.
	if len(actualChanges) != len(expectedChanges) {
		return false
	}

	// Index expected changes by path, because ordering is not guaranteed.
	pathToExpectedChange := make(map[string]*Change, len(expectedChanges))
	for _, expected := range expectedChanges {
		pathToExpectedChange[expected.Path] = expected
	}

	// Verify that the lists are equal.
	for _, actual := range actualChanges {
		// Look for the corresponding expected change. This also validates path
		// equivalence.
		expected, ok := pathToExpectedChange[actual.Path]
		if !ok {
			return false
		}

		// Verify that the old values match.
		if !actual.Old.Equal(expected.Old, true) {
			return false
		}

		// Verify that the new values match.
		if !actual.New.Equal(expected.New, true) {
			return false
		}
	}

	// At this point, the changes lists must be equivalent.
	return true
}

// testingConflictListsEqual determines whether or not two lists of conflicts
// are equivalent. The conflict lists do not need to be in the same order, but
// they do need to be structurally equivalent (i.e. not having their changes
// composed differently).
func testingConflictListsEqual(actualConflicts, expectedConflicts []*Conflict) bool {
	// Verify that the number of conflicts is the same in each case.
	if len(actualConflicts) != len(expectedConflicts) {
		return false
	}

	// Index expected conflicts by root path, because ordering is not
	// guaranteed.
	pathToExpectedConflict := make(map[string]*Conflict, len(expectedConflicts))
	for _, expected := range expectedConflicts {
		pathToExpectedConflict[expected.Root] = expected
	}

	// Verify that the lists are equal.
	for _, actual := range actualConflicts {
		// Look for the corresponding expected conflict. This also validates
		// conflict root equivalence.
		expected, ok := pathToExpectedConflict[actual.Root]
		if !ok {
			return false
		}

		// Verify that alpha changes are equal.
		if !testingChangeListsEqual(actual.AlphaChanges, expected.AlphaChanges) {
			return false
		}

		// Verify that beta changes are equal.
		if !testingChangeListsEqual(actual.BetaChanges, expected.BetaChanges) {
			return false
		}
	}

	// At this point, the conflict lists must be equivalent.
	return true
}

// testingProblemListsEqual determines whether or not two lists of problems are
// equivalent. The problem lists do not need to be in the same order. Expected
// errors can specify a wildcard match ("*") to match any error message from a
// problem with the same path. This is useful in testing where error messages
// are platform-dependent.
func testingProblemListsEqual(actualProblems, expectedProblems []*Problem) bool {
	// Verify that the number of problems is the same in each case.
	if len(actualProblems) != len(expectedProblems) {
		return false
	}

	// Index expected problems by path, because ordering is not guaranteed.
	pathToExpectedProblem := make(map[string]*Problem, len(expectedProblems))
	for _, expected := range expectedProblems {
		pathToExpectedProblem[expected.Path] = expected
	}

	// Verify that the lists are equal.
	for _, actual := range actualProblems {
		// Look for the corresponding expected problem.
		expected, ok := pathToExpectedProblem[actual.Path]
		if !ok {
			return false
		}

		// Verify that the errors are equal, but allow wildcard matches for
		// tesing.
		if expected.Error != "*" && actual.Error != expected.Error {
			return false
		}
	}

	// At this point, the problem lists must be equivalent.
	return true
}
