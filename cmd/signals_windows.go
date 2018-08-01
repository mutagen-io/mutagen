package cmd

import (
	"os"
	"syscall"
)

// TerminationSignals are those signals which Mutagen considers to be requesting
// termination.
var TerminationSignals = []os.Signal{
	// SIGINT is the only POSIX signal supported by Go on Windows, but Ctrl-C is
	// all we really need there anyway. Just for the record though, it's not a
	// native OS thing on Windows, but rather emulation performed by Go in
	// console environments.
	syscall.SIGINT,
}
