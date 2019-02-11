package filesystem

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/pkg/errors"
)

// TestWatchModeUnmarshal tests that unmarshaling from a string specification
// succeeeds for WatchMode.
func TestWatchModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Text          string
		ExpectedMode  WatchMode
		ExpectFailure bool
	}{
		{"", WatchMode_WatchModeDefault, true},
		{"asdf", WatchMode_WatchModeDefault, true},
		{"portable", WatchMode_WatchModePortable, false},
		{"force-poll", WatchMode_WatchModeForcePoll, false},
		{"no-watch", WatchMode_WatchModeNoWatch, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var mode WatchMode
		if err := mode.UnmarshalText([]byte(testCase.Text)); err != nil {
			if !testCase.ExpectFailure {
				t.Errorf("unable to unmarshal text (%s): %s", testCase.Text, err)
			}
		} else if testCase.ExpectFailure {
			t.Error("unmarshaling succeeded unexpectedly for text:", testCase.Text)
		} else if mode != testCase.ExpectedMode {
			t.Errorf(
				"unmarshaled mode (%s) does not match expected (%s)",
				mode,
				testCase.ExpectedMode,
			)
		}
	}
}

// TestWatchModeSupported tests that WatchMode support detection works as
// expected.
func TestWatchModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode            WatchMode
		ExpectSupported bool
	}{
		{WatchMode_WatchModeDefault, false},
		{WatchMode_WatchModePortable, true},
		{WatchMode_WatchModeForcePoll, true},
		{WatchMode_WatchModeNoWatch, true},
		{(WatchMode_WatchModeNoWatch + 1), false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if supported := testCase.Mode.Supported(); supported != testCase.ExpectSupported {
			t.Errorf(
				"mode support status (%t) does not match expected (%t)",
				supported,
				testCase.ExpectSupported,
			)
		}
	}
}

// TestWatchModeDescription tests that WatchMode description generation works as
// expected.
func TestWatchModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode                WatchMode
		ExpectedDescription string
	}{
		{WatchMode_WatchModeDefault, "Default"},
		{WatchMode_WatchModePortable, "Portable"},
		{WatchMode_WatchModeForcePoll, "Force Poll"},
		{WatchMode_WatchModeNoWatch, "No Watch"},
		{(WatchMode_WatchModeNoWatch + 1), "Unknown"},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if description := testCase.Mode.Description(); description != testCase.ExpectedDescription {
			t.Errorf(
				"mode description (%s) does not match expected (%s)",
				description,
				testCase.ExpectedDescription,
			)
		}
	}
}

const (
	testWatchEstablishWait = time.Second
)

func testWatchCycle(path string, mode WatchMode) error {
	// Create a cancellable watch context and defer its cancellation.
	watchContext, watchCancel := context.WithCancel(context.Background())
	defer watchCancel()

	// Create a watch event channel.
	events := make(chan struct{}, 1)

	// Start watching in a separate Goroutine.
	go Watch(watchContext, path, events, mode, 1)

	// HACK: Wait long enough for the recursive watch to be established or the
	// initial polling to occur. The CI systems aren't as fast as things are
	// locally, so we have to be a little conservative.
	time.Sleep(testWatchEstablishWait)

	// Compute the test file path.
	testFilePath := filepath.Join(path, "file")

	// Create a file inside the directory and wait for an event.
	if err := WriteFileAtomic(testFilePath, []byte{}, 0600); err != nil {
		return errors.Wrap(err, "unable to create file")
	}
	<-events

	// Modify a file inside the directory and wait for an event.
	if err := WriteFileAtomic(testFilePath, []byte{0, 0}, 0600); err != nil {
		return errors.Wrap(err, "unable to modify file")
	}
	<-events

	// If we're not on Windows, test that we detect permissions changes.
	if runtime.GOOS != "windows" {
		if err := os.Chmod(testFilePath, 0700); err != nil {
			return errors.Wrap(err, "unable to change file permissions")
		}
		<-events
	}

	// Remove a file inside the directory and wait for an event.
	if err := os.Remove(testFilePath); err != nil {
		return errors.Wrap(err, "unable to remove file")
	}
	<-events

	// Success.
	return nil
}

func TestWatchPortable(t *testing.T) {
	// Create a temporary directory and defer its removal.
	directory, err := ioutil.TempDir("", "mutagen_filesystem_watch")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Run the test cycle.
	if err := testWatchCycle(directory, WatchMode_WatchModePortable); err != nil {
		t.Fatal("watch cycle test failed:", err)
	}
}

func TestWatchForcePoll(t *testing.T) {
	// Create a temporary directory and defer its removal.
	directory, err := ioutil.TempDir("", "mutagen_filesystem_watch")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Run the test cycle.
	if err := testWatchCycle(directory, WatchMode_WatchModeForcePoll); err != nil {
		t.Fatal("watch cycle test failed:", err)
	}
}
