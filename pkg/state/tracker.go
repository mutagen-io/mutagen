package state

import (
	"context"
	"errors"
	"sync"
)

// ErrTrackingTerminated indicates that tracking was terminated before a polling
// operation saw any changes.
var ErrTrackingTerminated = errors.New("tracking terminated")

// pollResponse is used to respond to a polling request within Tracker.
type pollResponse struct {
	// index is the index at the time of the response.
	index uint64
	// terminated indicates whether or not tracking was terminated at the time
	// of the response.
	terminated bool
}

// pollRequest represents a polling request within Tracker.
type pollRequest struct {
	// previousIndex is the previous index for which state information was seen.
	previousIndex uint64
	// responses is used to respond to the polling request. It must be buffered.
	responses chan<- pollResponse
}

// Tracker provides index-based state tracking using a condition variable.
type Tracker struct {
	// change is the condition variable used to track changes. It is used to
	// signal state changes to index and terminated. It is also used to
	// serialize and signal changes to pollRequests.
	change *sync.Cond
	// index is the current state index.
	// NOTE: In theory, we should track and handle overflow on this index, but
	// given that an update period of 1 nanosecond would only cause an overflow
	// after about 584 years, the possibility isn't hugely concerning.
	//
	// Moreover, the "failure" mode in the case of overflow is that a poller who
	// waited an entire overflow period before an additional state change check,
	// and then managed to hit when the index was exactly the same as their last
	// check, would have to wait for an additional state change before detecting
	// an update. Given the vanishingly small likelihood of both conditions,
	// along with the minimal consequences, it's not worth hauling around a ton
	// of overflow handling code. We do perform a minimal amount of overflow
	// handling code on this value, but that's just to maintain the meaning of 0
	// as a previous state index in the unlikely event of an overflow.
	index uint64
	// terminated indicates whether or not tracking has been terminated.
	terminated bool
	// pollRequests is the set of current pollers.
	pollRequests map[*pollRequest]bool
	// trackDone is closed to signal that the tracking loop has exited.
	trackDone chan struct{}
}

// NewTracker creates a new tracker instance with a state index of 1.
func NewTracker() *Tracker {
	// Creack the tracker.
	tracker := &Tracker{
		change:       sync.NewCond(&sync.Mutex{}),
		index:        1,
		pollRequests: make(map[*pollRequest]bool),
		trackDone:    make(chan struct{}),
	}

	// Start the tracking loop.
	go tracker.track()

	// Done.
	return tracker
}

// track is the tracking loop entry point. It serves as a bridge between the
// world of condition variables and the world of channels.
func (t *Tracker) track() {
	// Defer closure of the tracking loop termination channel.
	defer close(t.trackDone)

	// Acquire the state lock and defer its release.
	t.change.L.Lock()
	defer t.change.L.Unlock()

	// Loop until terminated.
	for {
		// Check for and handle termination.
		if t.terminated {
			response := pollResponse{t.index, true}
			for r := range t.pollRequests {
				r.responses <- response
				delete(t.pollRequests, r)
			}
			return
		}

		// Signal any completed polling requests.
		// TODO: It would be nice if we had a better data structure where
		// iteration wasn't O(n) in the number of registered poll requests. It
		// feels like we could leverage the fact that index is monotonically
		// increasing and maybe use a heap (ordered by requests' previous
		// indices) to reduce the iteration overhead here, but it's not
		// performance critical for now. Such a design might motivate better
		// overflow handling as well. In any case, given that we're no longer
		// using sync.Cond.Broadcast, we're already saving O(n) iteration in the
		// Go runtime, so this is a reasonable tradeoff.
		for r := range t.pollRequests {
			if r.previousIndex != t.index {
				r.responses <- pollResponse{t.index, false}
				delete(t.pollRequests, r)
			}
		}

		// Wait for a state change.
		t.change.Wait()
	}
}

// Terminate terminates tracking.
func (t *Tracker) Terminate() {
	// Acquire the state lock.
	t.change.L.Lock()

	// Mark tracking as terminated.
	t.terminated = true

	// Signal to the tracking loop that termination has occurred.
	t.change.Signal()

	// Release the state lock.
	t.change.L.Unlock()

	// Wait for the tracking loop to exit.
	<-t.trackDone
}

// NotifyOfChange indicates the state index and notifies waiters.
func (t *Tracker) NotifyOfChange() {
	// Acquire the state lock and defer its release.
	t.change.L.Lock()
	defer t.change.L.Unlock()

	// Increment the state index. If we do overflow, then at least set the index
	// back to 1, because we want 0 to remain the sentinel value that returns an
	// immediate read of the current state index.
	t.index++
	if t.index == 0 {
		t.index = 1
	}

	// Signal the tracking loop.
	t.change.Signal()
}

// WaitForChange polls for a state index change from the specified previous
// index. It returns the new index at which the change was seen. If tracking is
// terminated before the polling operation completes, then the current state
// index is returned along with ErrTrackingTerminated. If the provided context
// is cancelled before the polling operation completes, then the current state
// index is returned along with context.Canceled. If a previous state index of 0
// is provided, then the current state index (which will always be greater than
// 0) is returned immediately.
func (t *Tracker) WaitForChange(ctx context.Context, previousIndex uint64) (uint64, error) {
	// If the previous index is 0, then an immediate read is being requested. In
	// that case we can just bypass the polling mechanism.
	if previousIndex == 0 {
		t.change.L.Lock()
		defer t.change.L.Unlock()
		if t.terminated {
			return t.index, ErrTrackingTerminated
		}
		return t.index, nil
	}

	// Acquire the state lock.
	t.change.L.Lock()

	// If tracking has already been terminated, then abort immediately because
	// polling won't function.
	if t.terminated {
		defer t.change.L.Unlock()
		return t.index, ErrTrackingTerminated
	}

	// Create and register the polling request.
	responses := make(chan pollResponse, 1)
	request := &pollRequest{previousIndex, responses}
	t.pollRequests[request] = true

	// Signal to the tracking loop that a new request has been registered.
	t.change.Signal()

	// Release the state lock.
	t.change.L.Unlock()

	// Wait for a state change or cancellation. If the request is cancelled,
	// then we'll deregister it ourselves (in which case there's no need to
	// notify the tracking loop). If the polling operation succeeds, then the
	// tracking loop will deregister the request.
	select {
	case <-ctx.Done():
		t.change.L.Lock()
		delete(t.pollRequests, request)
		defer t.change.L.Unlock()
		return t.index, context.Canceled
	case response := <-responses:
		if response.terminated {
			return response.index, ErrTrackingTerminated
		}
		return response.index, nil
	}
}
