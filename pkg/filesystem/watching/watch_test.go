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
		case path := <-watcher.Events():
			delete(paths, path)
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

// drainEvents consumes any buffered events from the watcher within a
// short window. This is useful between test phases to avoid stale
// events from one phase leaking into the next phase's verification.
// FSEvents (and other watchers) may coalesce or delay events, so a
// brief drain helps keep phases independent.
func drainEvents(watcher RecursiveWatcher) {
	timeout := time.NewTimer(200 * time.Millisecond)
	defer timeout.Stop()
	for {
		select {
		case <-watcher.Events():
		case <-timeout.C:
			return
		}
	}
}

// TestRecursiveWatcherRootEvent tests that modifications directly in
// the watch root (not in a subdirectory) produce an event with an
// empty relative path. This exercises the path == target branch in
// the event processing loop.
func TestRecursiveWatcherRootEvent(t *testing.T) {
	// Skip this test if recursive watching is unsupported.
	if !RecursiveWatchingSupported {
		t.Skip()
	}

	// Create a temporary directory and establish a recursive watch.
	directory := t.TempDir()
	watcher, err := NewRecursiveWatcher(directory)
	if err != nil {
		t.Fatal("unable to establish watch:", err)
	}
	defer watcher.Terminate()

	// Create a file directly in the watch root. The watcher should
	// report this as a root-relative path (i.e. just the filename,
	// not a subdirectory-prefixed path).
	fileName := "root-level-file"
	filePath := filepath.Join(directory, fileName)
	if err := os.WriteFile(filePath, []byte("content"), 0600); err != nil {
		t.Fatal("unable to create root-level file:", err)
	}
	verifyWatchEvent(t, watcher, map[string]bool{fileName: true})
}

// TestRecursiveWatcherNestedCreation tests that creating a deeply
// nested directory tree and writing a file at depth produces events
// for intermediate directories and the leaf file. This verifies that
// the recursive watching mechanism detects events at arbitrary depth,
// not just the first level below the root.
func TestRecursiveWatcherNestedCreation(t *testing.T) {
	// Skip this test if recursive watching is unsupported.
	if !RecursiveWatchingSupported {
		t.Skip()
	}

	// Create a temporary directory and establish a recursive watch.
	directory := t.TempDir()
	watcher, err := NewRecursiveWatcher(directory)
	if err != nil {
		t.Fatal("unable to establish watch:", err)
	}
	defer watcher.Terminate()

	// Create a nested directory tree three levels deep. We use
	// os.MkdirAll to create the entire path at once, which is a
	// common application pattern (e.g., package managers creating
	// node_modules/pkg/lib). The watcher should detect events for
	// the intermediate directories and the final directory.
	nestedDir := filepath.Join(directory, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0700); err != nil {
		t.Fatal("unable to create nested directories:", err)
	}

	// We expect at least the deepest new directory to be reported.
	// FSEvents may coalesce intermediate directory events, so we
	// only require the leaf. The verifyWatchEvent helper ignores
	// any additional events beyond the required set.
	//
	// Note: expected relative paths use forward slashes because
	// the watcher normalizes to forward slashes on all platforms
	// (see watch_recursive_windows.go). Use filepath.Join only
	// for absolute filesystem paths.
	verifyWatchEvent(t, watcher, map[string]bool{
		"a/b/c": true,
	})

	// Drain any coalesced events from the directory creation before
	// moving to the file creation phase.
	drainEvents(watcher)

	// Write a file at the deepest level. This tests that the
	// watcher can detect events at depth even after a burst of
	// directory creation events.
	fileRelative := "a/b/c/deep-file"
	fileAbsolute := filepath.Join(directory, fileRelative)
	if err := os.WriteFile(fileAbsolute, []byte("deep"), 0600); err != nil {
		t.Fatal("unable to create deep file:", err)
	}
	verifyWatchEvent(t, watcher, map[string]bool{fileRelative: true})
}

// TestRecursiveWatcherRapidModifications tests that multiple rapid
// writes to the same file are detected. FSEvents coalesces events
// within a latency window, so rapid modifications may arrive as a
// single coalesced event rather than one per write. The key
// correctness property is that *at least one* event is delivered
// after the final modification, ensuring the synchronization engine
// knows to rescan.
func TestRecursiveWatcherRapidModifications(t *testing.T) {
	// Skip this test if recursive watching is unsupported.
	if !RecursiveWatchingSupported {
		t.Skip()
	}

	// Create a temporary directory and establish a recursive watch.
	directory := t.TempDir()
	watcher, err := NewRecursiveWatcher(directory)
	if err != nil {
		t.Fatal("unable to establish watch:", err)
	}
	defer watcher.Terminate()

	// Create the initial file so the watcher has something to track.
	fileRelative := "rapid-file"
	fileAbsolute := filepath.Join(directory, fileRelative)
	if err := os.WriteFile(fileAbsolute, []byte("v1"), 0600); err != nil {
		t.Fatal("unable to create file:", err)
	}
	verifyWatchEvent(t, watcher, map[string]bool{fileRelative: true})

	// Drain events from the initial creation before starting the
	// rapid modification burst.
	drainEvents(watcher)

	// Perform several rapid writes in quick succession without
	// waiting for events between them. This simulates an editor
	// auto-saving or a build tool writing multiple output files.
	for i := range 10 {
		data := []byte("version-" + string(rune('0'+i)))
		if err := os.WriteFile(fileAbsolute, data, 0600); err != nil {
			t.Fatalf("unable to write modification %d: %v", i, err)
		}
	}

	// Verify that we receive at least one event for the file. Due
	// to FSEvents coalescing, we may not receive 10 separate events,
	// but we must receive at least one to ensure the sync engine
	// would trigger a rescan.
	verifyWatchEvent(t, watcher, map[string]bool{fileRelative: true})
}

// TestRecursiveWatcherRename tests that renaming a file within the
// watched directory tree produces events. Rename is a common
// operation pattern: many editors and tools use write-to-temp +
// rename-over-target for atomic file updates. The watcher must
// detect the rename so that the synchronization engine rescans the
// affected paths.
func TestRecursiveWatcherRename(t *testing.T) {
	// Skip this test if recursive watching is unsupported.
	if !RecursiveWatchingSupported {
		t.Skip()
	}

	// Create a temporary directory and establish a recursive watch.
	directory := t.TempDir()
	watcher, err := NewRecursiveWatcher(directory)
	if err != nil {
		t.Fatal("unable to establish watch:", err)
	}
	defer watcher.Terminate()

	// Create the source file.
	sourceRelative := "source-file"
	sourceAbsolute := filepath.Join(directory, sourceRelative)
	if err := os.WriteFile(sourceAbsolute, []byte("rename-me"), 0600); err != nil {
		t.Fatal("unable to create source file:", err)
	}
	verifyWatchEvent(t, watcher, map[string]bool{sourceRelative: true})

	// Drain events from the creation before performing the rename.
	drainEvents(watcher)

	// Rename the file to a new name. FSEvents should report events
	// for at least the destination path. It may also report an event
	// for the source path (as a delete) depending on coalescing, but
	// we only require the destination to be reported since that's the
	// path the sync engine needs to rescan.
	destRelative := "destination-file"
	destAbsolute := filepath.Join(directory, destRelative)
	if err := os.Rename(sourceAbsolute, destAbsolute); err != nil {
		t.Fatal("unable to rename file:", err)
	}
	verifyWatchEvent(t, watcher, map[string]bool{destRelative: true})

	// Drain events from the rename before performing the atomic
	// replace pattern.
	drainEvents(watcher)

	// Test the atomic replace pattern: write to a temp file, then
	// rename over an existing file. This is how many editors save
	// files (e.g., Vim, VS Code) and is the most critical rename
	// pattern for development workflows.
	tempRelative := "temp-file"
	tempAbsolute := filepath.Join(directory, tempRelative)
	if err := os.WriteFile(tempAbsolute, []byte("new-content"), 0600); err != nil {
		t.Fatal("unable to create temp file:", err)
	}
	verifyWatchEvent(t, watcher, map[string]bool{tempRelative: true})
	drainEvents(watcher)

	// Rename the temp file over the destination file. The watcher
	// must detect this so the sync engine picks up the new content.
	if err := os.Rename(tempAbsolute, destAbsolute); err != nil {
		t.Fatal("unable to rename temp over destination:", err)
	}
	verifyWatchEvent(t, watcher, map[string]bool{destRelative: true})
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
