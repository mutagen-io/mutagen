package state

import (
	"sync"
)

type TrackingLock struct {
	lock    sync.Mutex
	tracker *Tracker
}

func NewTrackingLock(tracker *Tracker) *TrackingLock {
	return &TrackingLock{
		tracker: tracker,
	}
}

func (l *TrackingLock) Lock() {
	l.lock.Lock()
}

func (l *TrackingLock) Unlock() {
	l.lock.Unlock()
	l.tracker.NotifyOfChange()
}

func (l *TrackingLock) UnlockWithoutNotify() {
	l.lock.Unlock()
}
