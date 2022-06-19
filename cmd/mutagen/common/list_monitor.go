package common

import (
	"time"
)

// MinimumMonitorUpdateInterval is the minimum interval between state updates in
// monitor commands (for both formatted status lines and templated output). It
// is designed to keep command line output visibly snappy whilst drastically
// reducing CPU load on both the monitor command and the daemon.
const MinimumMonitorUpdateInterval = 50 * time.Millisecond

// SessionDisplayMode encodes the mode in which session information should be
// displayed by session listing and monitoring functions.
type SessionDisplayMode uint8

const (
	// SessionDisplayModeList indicates that session information should be
	// displayed in list mode.
	SessionDisplayModeList SessionDisplayMode = iota
	// SessionDisplayModeListLong indicates that session information should be
	// displayed in list mode with extended details.
	SessionDisplayModeListLong
	// SessionDisplayModeMonitor indicates that session information should be
	// displayed in monitor mode.
	SessionDisplayModeMonitor
	// SessionDisplayModeMonitorLong indicates that session information should
	// be displayed in monitor mode with extended details.
	SessionDisplayModeMonitorLong
)

// FormatConnectionStatus formats a connection status for display.
func FormatConnectionStatus(connected bool) string {
	if connected {
		return "Yes"
	}
	return "No"
}
