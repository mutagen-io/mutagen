package state

import (
	"context"
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
		firstState, err := tracker.WaitForChange(context.Background(), 1)
		if err != nil || firstState != 2 {
			handoff <- false
			return
		}
		handoff <- true

		// Wait for termination and ensure that the state doesn't change.
		finalState, err := tracker.WaitForChange(context.Background(), firstState)
		handoff <- (finalState == firstState && err == ErrTrackingTerminated)
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

	// Acquire and release the lock in a way that won't change the state. We
	// don't expect a response here, but our termination response will be
	// invalid if this does change the state.
	lock.Lock()
	lock.UnlockWithoutNotify()

	// Terminate tracking and wait for a response.
	tracker.Terminate()
	select {
	case value := <-handoff:
		if !value {
			t.Fatal("received failure on tracking termination")
		}
	case <-time.After(trackerTestTimeout):
		t.Fatal("timeout failure on tracking termination")
	}
}
