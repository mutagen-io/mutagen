package mutagen

import (
	"os"
)

// DevelopmentModeEnabled controls whether or not development mode is enabled
// for Mutagen. It is set automatically based on the MUTAGEN_DEVELOPMENT
// environment variable.
var DevelopmentModeEnabled bool

func init() {
	// Check whether or not debugging should be enabled.
	DevelopmentModeEnabled = os.Getenv("MUTAGEN_DEVELOPMENT") == "1"
}
