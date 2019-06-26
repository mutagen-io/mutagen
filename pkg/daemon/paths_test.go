package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSubpath test that subpath succeeds and creates the daemon subdirectory.
func TestSubpath(t *testing.T) {
	// Compute a random subpath.
	path, err := subpath("something")
	if err != nil {
		t.Fatal("unable to compute subpath:", err)
	}

	// Ensure that the daemon subdirectory has been created.
	if s, err := os.Lstat(filepath.Dir(path)); err != nil {
		t.Fatal("unable to verify that daemon subdirectory exists:", err)
	} else if !s.IsDir() {
		t.Error("daemon subdirectory is not a directory")
	}
}

// TestIPCEndpointPath tests that IPCEndpointPath succeeds.
func TestIPCEndpointPath(t *testing.T) {
	if endpoint, err := IPCEndpointPath(); err != nil {
		t.Fatal("unable to compute IPC endpoint path:", err)
	} else if endpoint == "" {
		t.Error("empty IPC endpoint path returned")
	}
}
