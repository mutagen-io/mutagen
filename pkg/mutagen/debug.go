package mutagen

import (
	"os"
)

// DebugEnabled controls whether or not debugging is enabled for Mutagen. It is
// set automatically based on the MUTAGEN_DEBUG environment variable.
var DebugEnabled bool

func init() {
	// Check whether or not debugging should be enabled.
	DebugEnabled = os.Getenv("MUTAGEN_DEBUG") == "1"
}
