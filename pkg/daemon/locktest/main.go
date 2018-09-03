package main

import (
	"fmt"
	"os"

	"github.com/havoc-io/mutagen/pkg/daemon"
)

func main() {
	// Attempt to acquire the daemon lock and release it.
	if lock, err := daemon.AcquireLock(); err != nil {
		fmt.Fprintln(os.Stderr, "Mutagen lock acquisition failed")
		os.Exit(1)
	} else {
		lock.Unlock()
	}
}
