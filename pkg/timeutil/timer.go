package timeutil

import (
	"time"
)

// StopAndDrainTimer stops a timer and performs a non-blocking drain on its
// channel. This allows a timer to be stopped and drained without any knowledge
// of its current state.
func StopAndDrainTimer(timer *time.Timer) {
	timer.Stop()
	select {
	case <-timer.C:
	default:
	}
}
