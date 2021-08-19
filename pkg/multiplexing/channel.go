package multiplexing

import (
	"time"
)

// isClosed checks if a signaling channel is closed.
func isClosed(channel <-chan struct{}) bool {
	select {
	case <-channel:
		return true
	default:
		return false
	}
}

// wasPopulatedWithTime checks if a time signaling channel was populated with a
// time value and drains it if so.
func wasPopulatedWithTime(channel <-chan time.Time) bool {
	select {
	case <-channel:
		return true
	default:
		return false
	}
}
