package filesystem

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchModeUnmarshalPortable(t *testing.T) {
	var mode WatchMode
	if err := mode.UnmarshalText([]byte("portable")); err != nil {
		t.Fatal("unable to unmarshal text:", err)
	} else if mode != WatchMode_WatchPortable {
		t.Error("unmarshalled mode does not match expected")
	}
}

func TestWatchModeUnmarshalPOSIXRaw(t *testing.T) {
	var mode WatchMode
	if err := mode.UnmarshalText([]byte("force-poll")); err != nil {
		t.Fatal("unable to unmarshal text:", err)
	} else if mode != WatchMode_WatchForcePoll {
		t.Error("unmarshalled mode does not match expected")
	}
}

func TestWatchModeUnmarshalEmpty(t *testing.T) {
	var mode WatchMode
	if mode.UnmarshalText([]byte("")) == nil {
		t.Error("empty watch mode successfully unmarshalled")
	}
}

func TestWatchModeUnmarshalInvalid(t *testing.T) {
	var mode WatchMode
	if mode.UnmarshalText([]byte("invalid")) == nil {
		t.Error("invalid watch mode successfully unmarshalled")
	}
}

func TestWatchModeSupported(t *testing.T) {
	if WatchMode_WatchDefault.Supported() {
		t.Error("default watch mode considered supported")
	}
	if !WatchMode_WatchPortable.Supported() {
		t.Error("portable watch mode considered unsupported")
	}
	if !WatchMode_WatchForcePoll.Supported() {
		t.Error("force poll watch mode considered unsupported")
	}
	if (WatchMode_WatchForcePoll + 1).Supported() {
		t.Error("invalid watch mode considered supported")
	}
}

func TestWatchModeDescription(t *testing.T) {
	if description := WatchMode_WatchDefault.Description(); description != "Default" {
		t.Error("default watch mode description incorrect:", description, "!=", "Default")
	}
	if description := WatchMode_WatchPortable.Description(); description != "Portable" {
		t.Error("watch mode portable description incorrect:", description, "!=", "Portable")
	}
	if description := WatchMode_WatchForcePoll.Description(); description != "Force Poll" {
		t.Error("watch mode force poll description incorrect:", description, "!=", "Force Poll")
	}
	if description := (WatchMode_WatchForcePoll + 1).Description(); description != "Unknown" {
		t.Error("invalid watch mode description incorrect:", description, "!=", "Unknown")
	}
}

const (
	testWatchEstablishWait = 5 * time.Second
	testWatchTimeout       = 10 * time.Second
)

func TestWatchPortable(t *testing.T) {
	// Create a temporary directory in a subpath of the home directory and defer
	// its removal.
	directory, err := ioutil.TempDir(HomeDirectory, "mutagen_filesystem_watch")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Create a cancellable watch context.
	watchContext, watchCancel := context.WithCancel(context.Background())

	// Create a watch event channel.
	events := make(chan struct{}, 1)

	// Start watching in a separate Goroutine.
	go Watch(watchContext, directory, events, WatchMode_WatchPortable, 1)

	// HACK: Wait long enough for the recursive watch to be established or the
	// initial polling to occur. The CI systems aren't as fast as things are
	// locally, so we have to be a little conservative.
	time.Sleep(testWatchEstablishWait)

	// Create a file inside the directory.
	if err := WriteFileAtomic(filepath.Join(directory, "file"), []byte{}, 0600); err != nil {
		watchCancel()
		t.Fatal("unable to create file")
	}

	// Wait for an event or timeout.
	select {
	case <-events:
	case <-time.After(testWatchTimeout):
		t.Error("event not received in time")
	}

	// Cancel the watch.
	watchCancel()
}

func TestWatchForcePoll(t *testing.T) {
	// Create a temporary directory and defer its removal.
	directory, err := ioutil.TempDir("", "mutagen_filesystem_watch")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Create a cancellable watch context.
	watchContext, watchCancel := context.WithCancel(context.Background())

	// Create a watch event channel.
	events := make(chan struct{}, 1)

	// Start watching in a separate Goroutine, polling at 1 second intervals.
	go Watch(watchContext, directory, events, WatchMode_WatchForcePoll, 1)

	// HACK: Wait long enough for the initial polling to occur. The CI systems
	// aren't as fast as things are locally, so we have to be a little
	// conservative.
	time.Sleep(testWatchEstablishWait)

	// Create a file inside the directory.
	if err := WriteFileAtomic(filepath.Join(directory, "file"), []byte{}, 0600); err != nil {
		watchCancel()
		t.Fatal("unable to create file")
	}

	// Wait for an event or timeout.
	select {
	case <-events:
	case <-time.After(testWatchTimeout):
		t.Error("event not received in time")
	}

	// Cancel the watch.
	watchCancel()
}
