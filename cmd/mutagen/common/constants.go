package common

import (
	"time"
)

// MinimumMonitorUpdateInterval is the minimum interval between state updates in
// monitor commands (for both formatted status lines and templated output). It
// is designed to keep command line output visibly snappy whilst drastically
// reducing CPU load on both the monitor command and the daemon.
const MinimumMonitorUpdateInterval = 50 * time.Millisecond
