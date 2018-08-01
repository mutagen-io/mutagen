package state

import (
	"sync"
)

// TrackingLock provides locking facilities with automatic state tracking
// notifications.
type TrackingLock struct {
	// lock is the underlying mutex.
	lock sync.Mutex
	// tracker is the underlying tracker.
	tracker *Tracker
}

// NewTrackingLock creates a new tracking lock with the specified tracker.
func NewTrackingLock(tracker *Tracker) *TrackingLock {
	return &TrackingLock{
		tracker: tracker,
	}
}

// Lock locks the tracking lock.
func (l *TrackingLock) Lock() {
	l.lock.Lock()
}

// Unlock unlocks the tracking lock and triggers a state update notification.
func (l *TrackingLock) Unlock() {
	l.lock.Unlock()
	l.tracker.NotifyOfChange()
}

// UnlockWithoutNotify unlocks the tracking lock without triggering a state
// update notification.
func (l *TrackingLock) UnlockWithoutNotify() {
	l.lock.Unlock()
}
