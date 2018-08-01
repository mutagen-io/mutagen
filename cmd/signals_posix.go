// +build !windows

package cmd

import (
	"os"
	"syscall"
)

// TerminationSignals are those signals which Mutagen considers to be requesting
// termination.
var TerminationSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
	// TODO: We may want to consider expanding this list.
}
