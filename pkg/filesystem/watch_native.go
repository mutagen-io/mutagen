package filesystem

import (
	"time"
)

const (
	// watchNativeCoalescingWindow is the coalescing window for native watch
	// events.
	watchNativeCoalescingWindow = 10 * time.Millisecond
)
