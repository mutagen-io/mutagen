package state

import (
	"testing"
	"time"
)

func TestTrackingLock(t *testing.T) {
	// Create a tracker.
	tracker := NewTracker()

	// Wrap it in a tracking lock.
	lock := NewTrackingLock(tracker)

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

	// Acquire and release the lock in a way that will change the state, and
	// then wait for a response.
	lock.Lock()
	lock.Unlock()
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

	// Acquire and release the lock in a way that won't change the state. We
	// don't expect a response here, but our poison response will be invalid if
	// this does change the state.
	lock.Lock()
	lock.UnlockWithoutNotify()

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
