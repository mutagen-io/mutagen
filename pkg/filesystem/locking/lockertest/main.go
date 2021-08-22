package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/pkg/filesystem/locking"
)

func main() {
	// Validate arguments and extract the lock path.
	if len(os.Args) != 2 {
		cmd.Fatal(errors.New("invalid number of arguments"))
	} else if os.Args[1] == "" {
		cmd.Fatal(errors.New("empty lock path"))
	}
	path := os.Args[1]

	// Create a locker and attempt to acquire the lock.
	if locker, err := locking.NewLocker(path, 0600); err != nil {
		cmd.Fatal(errors.New("unable to create filesystem locker"))
	} else if err = locker.Lock(false); err != nil {
		cmd.Fatal(fmt.Errorf("lock acquisition failed: %w", err))
	} else if err = locker.Unlock(); err != nil {
		cmd.Fatal(fmt.Errorf("lock release failed: %w", err))
	} else if err = locker.Close(); err != nil {
		cmd.Fatal(fmt.Errorf("locker closure failed: %w", err))
	}
}
