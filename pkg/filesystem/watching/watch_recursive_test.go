package watching

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

const (
	// timeBetweenOperations is the time window to wait between file operations
	// in TestRecursiveWatchCycle. This needs to be large enough that multiple
	// file operations are coalesced into a single notification.
	timeBetweenOperations = 100 * time.Millisecond

	// maximumEventWaitTime is the maximum amount of time that
	// TestRecursiveWatchCycle will wait for an event to come in.
	maximumEventWaitTime = 5 * time.Second
)

// TestRecursiveWatchCycle tests WatchRecursive with a simple set of filesystem
// operations. It's not an exhaustive exercise of the watching code, more of a
// litmus test.
func TestRecursiveWatchCycle(t *testing.T) {
	// If this platform doesn't support recursive watching, then skip this test.
	if !RecursiveWatchingSupported {
		t.Skip()
	}

	// Create a temporary directory and defer its removal.
	directory, err := ioutil.TempDir("", "mutagen_filesystem_watch")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Create a cancellable watch context and defer its cancellation.
	watchContext, watchCancel := context.WithCancel(context.Background())
	defer watchCancel()

	// Create a watch event channel. We only check events at the end, so we need
	// to make sure it's large enough to handle anything we might generate.
	events := make(chan string, 50)

	// Start watching in a separate Goroutine, watching for errors.
	watchErrors := make(chan error, 1)
	go func() {
		watchErrors <- WatchRecursive(watchContext, directory, events)
	}()

	// Wait for the initial strobe event which indicates watch initialization.
	if strobe, ok := <-events; !ok {
		t.Fatal("events channel closed unexpectedly")
	} else if strobe != "" {
		t.Fatal("strobe event had incorrect path")
	}

	// Compute the test file path.
	testFileName := "test_file"
	testFilePath := filepath.Join(directory, testFileName)

	// Track how many events we expect for the test file.
	expectedTestFileEventCount := 0

	// Create a file inside the directory and wait for an event.
	file, err := os.Create(testFilePath)
	if err != nil {
		t.Fatal("unable to create test file:", err)
	}
	file.Close()
	expectedTestFileEventCount++

	// Wait before performing another operation.
	time.Sleep(timeBetweenOperations)

	// Modify the test file.
	file, err = os.OpenFile(testFilePath, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal("unable to open test file:", err)
	} else if _, err = file.Write([]byte("data")); err != nil {
		t.Error("unable to write data to file:", err)
	}
	file.Close()
	expectedTestFileEventCount++

	// Wait before performing another operation.
	time.Sleep(timeBetweenOperations)

	// If we're not on Windows, test that we detect permissions changes.
	if runtime.GOOS != "windows" {
		// Perform the change and update the expected event count.
		if err := os.Chmod(testFilePath, 0700); err != nil {
			t.Fatal("unable to change file permissions:", err)
		}
		expectedTestFileEventCount++

		// Wait before performing another operation.
		time.Sleep(timeBetweenOperations)
	}

	// Remove the test file.
	if err := os.Remove(testFilePath); err != nil {
		t.Fatal("unable to remove test file:", err)
	}
	expectedTestFileEventCount++

	// Create a timer that will govern our maximum event wait time and defer its
	// termination (in case it's still running).
	deadlineTimer := time.NewTimer(maximumEventWaitTime)
	defer deadlineTimer.Stop()

	// Loop over events, ensuring that we see events for the test file the
	// expected number of times, and that all other event paths (if any) are the
	// root path (which, depending on the platform, may show updates since when
	// it's modified by the addition and/or removal of the file). Also ensure
	// that these events come in before the deadline and that we don't see any
	// watch errors.
Verification:
	for {
		select {
		case <-deadlineTimer.C:
			t.Fatal("events not received in time")
		case err := <-watchErrors:
			t.Fatal("watch error:", err)
		case path, ok := <-events:
			// Watch for event channel closure. If we see this, then we know
			// that a watch error is coming.
			if !ok {
				t.Fatal("events channel closed unexpectedly, received watch error:", <-watchErrors)
			}

			// Verified that the path is an allowed value.
			if path == "" {
				continue
			} else if path != testFileName {
				t.Fatal("saw unexpected event path:", path)
			}

			// Update the event count.
			expectedTestFileEventCount--
			if expectedTestFileEventCount == 0 {
				break Verification
			}
		}
	}
}
