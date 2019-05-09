package watching

import (
	"time"
)

const (
	// watchNativeEventsBufferSize is the event buffer size to use for raw
	// native events.
	watchNativeEventsBufferSize = 25
	// watchNativeCoalescingWindow is the coalescing window for native watch
	// events.
	watchNativeCoalescingWindow = 10 * time.Millisecond
)
