package remote

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureSynchronizationRootParentExistsCreatesMissingParentsOnly(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing", "parent", "root")

	if err := ensureSynchronizationRootParentExists(root); err != nil {
		t.Fatalf("unable to ensure parent hierarchy: %v", err)
	}

	if _, err := os.Stat(filepath.Dir(root)); err != nil {
		t.Fatalf("parent directory hierarchy not created: %v", err)
	}

	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("synchronization root should not be created directly, err: %v", err)
	}
}

func TestEnsureSynchronizationRootParentExistsNoopIfRootExists(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatalf("unable to create test root: %v", err)
	}

	if err := ensureSynchronizationRootParentExists(root); err != nil {
		t.Fatalf("unable to ensure parent hierarchy for existing root: %v", err)
	}
}
