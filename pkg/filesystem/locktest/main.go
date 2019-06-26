package main

import (
	"fmt"
	"os"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

func main() {
	// Attempt to acquire the Mutagen lock and release it.
	if locker, err := filesystem.AcquireMutagenLock(); err != nil {
		fmt.Fprintln(os.Stderr, "Mutagen lock acquisition failed")
		os.Exit(1)
	} else {
		locker.Close()
	}
}
