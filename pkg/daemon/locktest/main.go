package main

import (
	"os"

	"github.com/havoc-io/mutagen/pkg/daemon"
)

const (
	// exitCodeLockAcquireFail is a sentinel exit code used to indicate lock
	// acquisition failure.
	exitCodeLockAcquireFail = 64
)

func main() {
	// Attempt to acquire the daemon lock and release it.
	if lock, err := daemon.AcquireLock(); err != nil {
		os.Exit(exitCodeLockAcquireFail)
	} else {
		lock.Unlock()
	}
}
