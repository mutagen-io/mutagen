package daemon

import (
	"os"
)

// AutostartDisabled controls whether or not daemon autostart is disabled for
// Mutagen. It is set automatically based on the MUTAGEN_DISABLE_AUTOSTART
// environment variable.
var AutostartDisabled bool

func init() {
	// Check whether or not autostart should be disabled.
	AutostartDisabled = os.Getenv("MUTAGEN_DISABLE_AUTOSTART") == "1"
}
