package state

import (
	"sync"
)

type NotifyingLock struct {
	lock sync.Mutex
	notifier *Notifier
}

func NewNotifyingLock(notifier *Notifier) *NotifyingLock {
	return &NotifyingLock{
		notifier: notifier,
	}
}

func (n *NotifyingLock) Lock() {
	n.lock.Lock()
}

func (n *NotifyingLock) Unlock() {
	n.lock.Unlock()
	n.notifier.Notify()
}

func (n *NotifyingLock) UnlockWithoutNotify() {
	n.lock.Unlock()
}
