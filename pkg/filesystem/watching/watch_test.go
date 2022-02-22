package watching

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

const (
	// maximumEventWaitTime is the maximum amount of time that verifyWatchEvent
	// will wait for an event to be received.
	maximumEventWaitTime = 5 * time.Second
)

// verifyWatchEvent is a helper function to verify that events are received by a
// watcher. It accepts a RecursiveWatcher, but can also be used for a
// NonRecursiveWatcher (since the interface semantics are compatible). The paths
// map may be modified by this function and should thus not be reused.
func verifyWatchEvent(t *testing.T, watcher RecursiveWatcher, paths map[string]bool) {
	// Indicate that this is a helper function.
	t.Helper()

	// Create a deadline for event reception and ensure its cancellation.
	deadline := time.NewTimer(maximumEventWaitTime)
	defer deadline.Stop()

	// Perform the waiting operation.
	for len(paths) > 0 {
		select {
		case event := <-watcher.Events():
			for path := range paths {
				if event[path] {
					delete(paths, path)
				}
			}
		case err := <-watcher.Errors():
			t.Fatal("watcher error:", err)
		case <-deadline.C:
			t.Fatal("event reception deadline exceeded:", paths)
		}
	}
}

// TestRecursiveWatcher tests the platform's RecursiveWatcher implementation (if
// any) with a simple set of filesystem operations.
func TestRecursiveWatcher(t *testing.T) {
	// Skip this test if recursive watchig is unsupported.
	if !RecursiveWatchingSupported {
		t.Skip()
	}

	// Create a temporary directory (that will be automatically removed).
	directory := t.TempDir()

	// Create the watcher and defer its termination.
	watcher, err := NewRecursiveWatcher(directory)
	if err != nil {
		t.Fatal("unable to establish watch:", err)
	}
	defer watcher.Terminate()

	// Create a subdirectory.
	subdirectoryRelative := "subdirectory"
	subdirectoryAbsolute := filepath.Join(directory, subdirectoryRelative)
	if err := os.Mkdir(subdirectoryAbsolute, 0700); err != nil {
		t.Fatal("unable to create subdirectory:", err)
	}
	verifyWatchEvent(t, watcher, map[string]bool{subdirectoryRelative: true})

	// Create a file inside the subdirectory.
	fileRelative := "subdirectory/file"
	fileAbsolute := filepath.Join(directory, fileRelative)
	if err := os.WriteFile(fileAbsolute, nil, 0600); err != nil {
		t.Fatal("unable to create test file:", err)
	}
	verifyWatchEvent(t, watcher, map[string]bool{fileRelative: true})

	// Modify the test file.
	if err := os.WriteFile(fileAbsolute, []byte("data"), 0600); err != nil {
		t.Fatal("unable to modify test file:", err)
	}
	verifyWatchEvent(t, watcher, map[string]bool{fileRelative: true})

	// If we're not on Windows, test that we detect permissions changes.
	if runtime.GOOS != "windows" {
		if err := os.Chmod(fileAbsolute, 0700); err != nil {
			t.Fatal("unable to change file permissions:", err)
		}
		verifyWatchEvent(t, watcher, map[string]bool{fileRelative: true})
	}

	// Remove the test file.
	if err := os.Remove(fileAbsolute); err != nil {
		t.Fatal("unable to remove test file:", err)
	}
	verifyWatchEvent(t, watcher, map[string]bool{fileRelative: true})
}

// TestNonRecursiveWatcher tests the platform's NonRecursiveWatcher
// implementation (if any) with a simple set of filesystem operations.
func TestNonRecursiveWatcher(t *testing.T) {
	// Skip this test if non-recursive watchig is unsupported.
	if !NonRecursiveWatchingSupported {
		t.Skip()
	}

	// Create a temporary directory (that will be automatically removed).
	directory := t.TempDir()

	// Create the watcher and defer its termination.
	watcher, err := NewNonRecursiveWatcher()
	if err != nil {
		t.Fatal("unable to create watcher:", err)
	}
	watcher.Watch(directory)
	defer watcher.Terminate()

	// Create a subdirectory.
	subdirectoryPath := filepath.Join(directory, "subdirectory")
	if err := os.Mkdir(subdirectoryPath, 0700); err != nil {
		t.Fatal("unable to create subdirectory:", err)
	}
	verifyWatchEvent(t, watcher, map[string]bool{subdirectoryPath: true})

	// Create a file.
	filePath := filepath.Join(directory, "file")
	if err := os.WriteFile(filePath, nil, 0600); err != nil {
		t.Fatal("unable to create test file:", err)
	}
	verifyWatchEvent(t, watcher, map[string]bool{filePath: true})

	// Modify the test file.
	if err := os.WriteFile(filePath, []byte("data"), 0600); err != nil {
		t.Fatal("unable to modify test file:", err)
	}
	verifyWatchEvent(t, watcher, map[string]bool{filePath: true})

	// If we're not on Windows, test that we detect permissions changes.
	if runtime.GOOS != "windows" {
		if err := os.Chmod(filePath, 0700); err != nil {
			t.Fatal("unable to change file permissions:", err)
		}
		verifyWatchEvent(t, watcher, map[string]bool{filePath: true})
	}

	// Remove the test file.
	if err := os.Remove(filePath); err != nil {
		t.Fatal("unable to remove test file:", err)
	}
	verifyWatchEvent(t, watcher, map[string]bool{filePath: true})
}
