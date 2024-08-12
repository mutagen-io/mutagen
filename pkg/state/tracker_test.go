package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

// trackerTestTimeout prevents tracker tests from timing out. It also sets an
// indirect performance boundary on update detection time.
const trackerTestTimeout = 1 * time.Second

// TestTracker tests Tracker.
func TestTracker(t *testing.T) {
	// Create a tracker.
	tracker := NewTracker()

	// Create a channel for Goroutine communication.
	handoff := make(chan bool)

	// Create a cancellable context that we can use to tracking testing. Ensure
	// that it's cancelled by the time we return, just in case it isn't
	// cancelled during testing.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start a Goroutine with which we'll coordinate.
	go func() {
		// Wait indefinitely for a successful change from the initial tracker
		// state (1).
		firstState, err := tracker.WaitForChange(context.Background(), 1)
		if err != nil || firstState != 2 {
			handoff <- false
			return
		}
		handoff <- true

		// Perform a preempted wait and ensure that the state doesn't change.
		secondState, err := tracker.WaitForChange(ctx, firstState)
		if err != context.Canceled || secondState != firstState {
			handoff <- false
			return
		}
		handoff <- true

		// Wait for termination and ensure that the state doesn't change.
		finalState, err := tracker.WaitForChange(context.Background(), secondState)
		handoff <- (finalState == firstState && errors.Is(err, ErrTrackingTerminated))
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

	// Cancel the polling context and wait for a response.
	cancel()
	select {
	case value := <-handoff:
		if !value {
			t.Fatal("received failure on state tracking with cancellation")
		}
	case <-time.After(trackerTestTimeout):
		t.Fatal("timeout failure on state tracking with cancellation")
	}

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
