// +build !windows

package cmd

import (
	"os"
	"syscall"
)

// TODO: We may want to consider expanding this list.
var TerminationSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
}
