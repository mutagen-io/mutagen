package sync

import (
	"testing"
)

func TestReconcileNonDeletionChangesOnly(t *testing.T) {
	changes := []*Change{
		{
			Path: "file",
			New:  testFileEntry,
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
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(nil, nil, nil)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 0 {
		t.Error("alpha transitions non-empty")
	}

	// Validate beta transitions.
	if len(βTransitions) != 0 {
		t.Error("beta transitions non-empty")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileDirectoryNothingChanged(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		testDirectory1Entry,
		testDirectory1Entry,
		testDirectory1Entry,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 0 {
		t.Error("alpha transitions non-empty")
	}

	// Validate beta transitions.
	if len(βTransitions) != 0 {
		t.Error("beta transitions non-empty")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileFileNothingChanged(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		testFileEntry,
		testFileEntry,
		testFileEntry,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 0 {
		t.Error("alpha transitions non-empty")
	}

	// Validate beta transitions.
	if len(βTransitions) != 0 {
		t.Error("beta transitions non-empty")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileAlphaDeletedRoot(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		testDirectory1Entry,
		nil,
		testDirectory1Entry,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 0 {
		t.Error("alpha transitions non-empty")
	}

	// Validate beta transitions.
	if len(βTransitions) != 1 {
		t.Error("beta transitions have unexpected length")
	} else if βTransitions[0].Path != "" {
		t.Error("beta transition has unexpected path")
	} else if βTransitions[0].Old != testDirectory1Entry {
		t.Error("beta transition has unexpected old entry")
	} else if βTransitions[0].New != nil {
		t.Error("beta transition has unexpected new entry")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileBetaDeletedRoot(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		testDirectory1Entry,
		testDirectory1Entry,
		nil,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 1 {
		t.Error("alpha transitions have unexpected length")
	} else if αTransitions[0].Path != "" {
		t.Error("alpha transition has unexpected path")
	} else if αTransitions[0].Old != testDirectory1Entry {
		t.Error("alpha transition has unexpected old entry")
	} else if αTransitions[0].New != nil {
		t.Error("alpha transition has unexpected new entry")
	}

	// Validate beta transitions.
	if len(βTransitions) != 0 {
		t.Error("beta transitions non-empty")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileBothDeletedRoot(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		testDirectory1Entry,
		nil,
		nil,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 1 {
		t.Error("ancestor changes have unexpected length")
	} else if ancestorChanges[0].Path != "" {
		t.Error("ancestor change has unexpected path")
	} else if ancestorChanges[0].Old != nil {
		t.Error("ancestor change has unexpected old entry")
	} else if ancestorChanges[0].New != nil {
		t.Error("ancestor change has unexpected new entry")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 0 {
		t.Error("alpha transitions non-empty")
	}

	// Validate beta transitions.
	if len(βTransitions) != 0 {
		t.Error("beta transitions non-empty")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileAlphaCreatedRoot(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		nil,
		testFileEntry,
		nil,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 0 {
		t.Error("alpha transitions non-empty")
	}

	// Validate beta transitions.
	if len(βTransitions) != 1 {
		t.Error("beta transitions have unexpected length")
	} else if βTransitions[0].Path != "" {
		t.Error("beta transition has unexpected path")
	} else if βTransitions[0].Old != nil {
		t.Error("beta transition has unexpected old entry")
	} else if βTransitions[0].New != testFileEntry {
		t.Error("beta transition has unexpected new entry")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileBetaCreatedRoot(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		nil,
		nil,
		testFileEntry,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 1 {
		t.Error("alpha transitions have unexpected length")
	} else if αTransitions[0].Path != "" {
		t.Error("alpha transition has unexpected path")
	} else if αTransitions[0].Old != nil {
		t.Error("alpha transition has unexpected old entry")
	} else if αTransitions[0].New != testFileEntry {
		t.Error("alpha transition has unexpected new entry")
	}

	// Validate beta transitions.
	if len(βTransitions) != 0 {
		t.Error("beta transitions non-empty")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileBothCreatedSameFile(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		nil,
		testFileEntry,
		testFileEntry,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 1 {
		t.Error("unexpected number of ancestor changes")
	} else if newAncestor, err := Apply(nil, ancestorChanges); err != nil {
		t.Error("unable to apply ancestor changes")
	} else if !newAncestor.Equal(testFileEntry) {
		t.Error("post-change ancestor does not match expected")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 0 {
		t.Error("alpha transitions non-empty")
	}

	// Validate beta transitions.
	if len(βTransitions) != 0 {
		t.Error("beta transitions non-empty")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileBothCreatedSameDirectory(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		nil,
		testDirectory1Entry,
		testDirectory1Entry,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 11 {
		t.Error("unexpected number of ancestor changes")
	} else if newAncestor, err := Apply(nil, ancestorChanges); err != nil {
		t.Error("unable to apply ancestor changes")
	} else if !newAncestor.Equal(testDirectory1Entry) {
		t.Error("post-change ancestor does not match expected")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 0 {
		t.Error("alpha transitions non-empty")
	}

	// Validate beta transitions.
	if len(βTransitions) != 0 {
		t.Error("beta transitions non-empty")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileBothCreatedDifferentDirectory(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		nil,
		testDirectory1Entry,
		testDirectory2Entry,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 4 {
		t.Error("unexpected number of ancestor changes")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 4 {
		t.Error("unexpected number of alpha transitions")
	}

	// Validate beta transitions.
	if len(βTransitions) != 3 {
		t.Error("unexpected number of beta transitions")
	}

	// Validate conflicts.
	if len(conflicts) != 1 {
		t.Error("unexpected number of conflicts")
	}
}

func TestReconcileBothCreatedDifferentTypes(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		nil,
		testDirectory1Entry,
		testFileEntry,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 0 {
		t.Error("alpha transitions non-empty")
	}

	// Validate beta transitions.
	if len(βTransitions) != 0 {
		t.Error("beta transitions non-empty")
	}

	// Validate conflicts.
	if len(conflicts) != 1 {
		t.Error("unexpected number of conflicts")
	}
}

func TestReconcileAlphaDeletedRootBetaCreatedFile(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		testDirectory1Entry,
		nil,
		testFileEntry,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 1 {
		t.Error("alpha transitions have unexpected length")
	} else if αTransitions[0].Path != "" {
		t.Error("alpha transition has unexpected path")
	} else if αTransitions[0].Old != nil {
		t.Error("alpha transition has unexpected old entry")
	} else if αTransitions[0].New != testFileEntry {
		t.Error("alpha transition has unexpected new entry")
	}

	// Validate beta transitions.
	if len(βTransitions) != 0 {
		t.Error("beta transitions non-empty")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileAlphaCreatedFileBetaDeletedRoot(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		testDirectory1Entry,
		testFileEntry,
		nil,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 0 {
		t.Error("alpha transitions non-empty")
	}

	// Validate beta transitions.
	if len(βTransitions) != 1 {
		t.Error("beta transitions have unexpected length")
	} else if βTransitions[0].Path != "" {
		t.Error("beta transition has unexpected path")
	} else if βTransitions[0].Old != nil {
		t.Error("beta transition has unexpected old entry")
	} else if βTransitions[0].New != testFileEntry {
		t.Error("beta transition has unexpected new entry")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileAlphaDeletedRootBetaCreatedDirectory(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		testFileEntry,
		nil,
		testDirectory1Entry,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 1 {
		t.Error("alpha transitions have unexpected length")
	} else if αTransitions[0].Path != "" {
		t.Error("alpha transition has unexpected path")
	} else if αTransitions[0].Old != nil {
		t.Error("alpha transition has unexpected old entry")
	} else if αTransitions[0].New != testDirectory1Entry {
		t.Error("alpha transition has unexpected new entry")
	}

	// Validate beta transitions.
	if len(βTransitions) != 0 {
		t.Error("beta transitions non-empty")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileAlphaCreatedDirectoryBetaDeletedRoot(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		testFileEntry,
		testDirectory1Entry,
		nil,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 0 {
		t.Error("alpha transitions non-empty")
	}

	// Validate beta transitions.
	if len(βTransitions) != 1 {
		t.Error("beta transitions have unexpected length")
	} else if βTransitions[0].Path != "" {
		t.Error("beta transition has unexpected path")
	} else if βTransitions[0].Old != nil {
		t.Error("beta transition has unexpected old entry")
	} else if βTransitions[0].New != testDirectory1Entry {
		t.Error("beta transition has unexpected new entry")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileAlphaChangedDirectory(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		testDirectory2Entry,
		testDirectory3Entry,
		testDirectory2Entry,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 0 {
		t.Error("alpha transitions non-empty")
	}

	// Validate beta transitions.
	if len(βTransitions) != 2 {
		t.Error("beta transitions have unexpected length")
	} else if newBeta, err := Apply(testDirectory2Entry, βTransitions); err != nil {
		t.Error("unable to apply beta transitions")
	} else if !newBeta.Equal(testDirectory3Entry) {
		t.Error("post-transition beta does not match expected")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileBetaChangedDirectory(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		testDirectory3Entry,
		testDirectory3Entry,
		testDirectory2Entry,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 2 {
		t.Error("beta transitions have unexpected length")
	} else if newBeta, err := Apply(testDirectory3Entry, αTransitions); err != nil {
		t.Error("unable to apply beta transitions")
	} else if !newBeta.Equal(testDirectory2Entry) {
		t.Error("post-transition beta does not match expected")
	}

	// Validate beta transitions.
	if len(βTransitions) != 0 {
		t.Error("beta transitions non-empty")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileAlphaReplacedDirectoryBetaDeletedPartialContents(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		testDirectory2Entry,
		testFileEntry,
		testDirectory3Entry,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 0 {
		t.Error("alpha transitions non-empty")
	}

	// Validate beta transitions.
	if len(βTransitions) != 1 {
		t.Error("beta transitions have unexpected length")
	} else if βTransitions[0].Path != "" {
		t.Error("beta transition has unexpected path")
	} else if βTransitions[0].Old != testDirectory3Entry {
		t.Error("beta transition has unexpected old entry")
	} else if βTransitions[0].New != testFileEntry {
		t.Error("beta transition has unexpected new entry")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}

func TestReconcileAlphaDeletedPartialContentsBetaReplacedDirectory(t *testing.T) {
	// Perform reconciliation.
	ancestorChanges, αTransitions, βTransitions, conflicts := Reconcile(
		testDirectory2Entry,
		testDirectory3Entry,
		testFileEntry,
	)

	// Validate ancestor changes.
	if len(ancestorChanges) != 0 {
		t.Error("ancestor changes non-empty")
	}

	// Validate alpha transitions.
	if len(αTransitions) != 1 {
		t.Error("alpha transitions have unexpected length")
	} else if αTransitions[0].Path != "" {
		t.Error("alpha transition has unexpected path")
	} else if αTransitions[0].Old != testDirectory3Entry {
		t.Error("alpha transition has unexpected old entry")
	} else if αTransitions[0].New != testFileEntry {
		t.Error("alpha transition has unexpected new entry")
	}

	// Validate beta transitions.
	if len(βTransitions) != 0 {
		t.Error("beta transitions non-empty")
	}

	// Validate conflicts.
	if len(conflicts) != 0 {
		t.Error("conflicts present")
	}
}
