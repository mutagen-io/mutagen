package cmd

import (
	"os"
	"syscall"
)

// TerminationSignals are those signals which Mutagen considers to be requesting
// termination. Certain other signals that also request termination (such as
// SIGABRT) are intentionally ignored because they're handled by the Go runtime
// and have special behavior (such as dumping a stack trace). Both SIGINT and
// SIGTERM are emulated on Windows (SIGINT on Ctrl-C and Ctrl-Break and SIGTERM
// on CTRL_CLOSE_EVENT, CTRL_LOGOFF_EVENT, and CTRL_SHUTDOWN_EVENT).
var TerminationSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
}
