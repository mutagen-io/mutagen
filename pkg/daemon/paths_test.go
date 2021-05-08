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

// TestLockPath tests that lockPath succeeds.
func TestLockPath(t *testing.T) {
	if path, err := lockPath(); err != nil {
		t.Fatal("unable to compute lock path:", err)
	} else if path == "" {
		t.Error("empty lock path returned")
	}
}

// TestEndpointPath tests that EndpointPath succeeds.
func TestEndpointPath(t *testing.T) {
	if path, err := EndpointPath(); err != nil {
		t.Fatal("unable to compute IPC endpoint path:", err)
	} else if path == "" {
		t.Error("empty IPC endpoint path returned")
	}
}

// TestLogPath tests that logPath succeeds.
func TestLogPath(t *testing.T) {
	if path, err := logPath(); err != nil {
		t.Fatal("unable to compute log path:", err)
	} else if path == "" {
		t.Error("empty log path returned")
	}
}

// TestTokenPath tests that TokenPath succeeds.
func TestTokenPath(t *testing.T) {
	if path, err := TokenPath(); err != nil {
		t.Fatal("unable to compute token path:", err)
	} else if path == "" {
		t.Error("empty token path returned")
	}
}

// TestPortPath tests that PortPath succeeds.
func TestPortPath(t *testing.T) {
	if path, err := PortPath(); err != nil {
		t.Fatal("unable to compute port path:", err)
	} else if path == "" {
		t.Error("empty port path returned")
	}
}
