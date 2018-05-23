package state

import (
	"testing"
	"time"
)

const (
	trackerTestSleep   = 10 * time.Millisecond
	trackerTestTimeout = 1 * time.Second
)

func TestTracker(t *testing.T) {
	// Create a tracker.
	tracker := NewTracker()

	// Create a channel for Goroutine communication.
	handoff := make(chan bool)

	// Start a Goroutine with which we'll coordinate.
	go func() {
		// Wait for a successful change from the initial tracker state (1).
		firstState, poisoned := tracker.WaitForChange(1)
		if poisoned || firstState != 2 {
			handoff <- false
			return
		}
		handoff <- true

		// Wait for poisoning.
		_, poisoned = tracker.WaitForChange(firstState)
		handoff <- poisoned
	}()

	// Notify of a change and wait for a response.
	tracker.NotifyOfChange()
	select {
	case value := <-handoff:
		if !value {
			t.Fatal("received failure on state tracking")
		}
	case <-time.After(trackerTestTimeout):
		t.Fatal("timeout failure on state tracking")
	}

	// Sleep for enough time that the Goroutine can invoke the condition
	// variable wait.
	time.Sleep(trackerTestSleep)

	// Poison the tracker and wait for a response.
	tracker.Poison()
	select {
	case value := <-handoff:
		if !value {
			t.Fatal("received failure on state poisoning")
		}
	case <-time.After(trackerTestTimeout):
		t.Fatal("timeout failure on state poisoning")
	}
}
