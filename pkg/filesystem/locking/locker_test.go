package locking

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestLockerFailOnDirectory(t *testing.T) {
	// Create a temporary directory and defer its removal.
	directory, err := ioutil.TempDir("", "mutagen_filesystem_lock")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Ensure that locker creation fails.
	if _, err := NewLocker(directory, 0600); err == nil {
		t.Fatal("creating a locker on a directory path succeeded")
	}
}

func TestLockerCycle(t *testing.T) {
	// Create a temporary file and defer its removal.
	lockfile, err := ioutil.TempFile("", "mutagen_filesystem_lock")
	if err != nil {
		t.Fatal("unable to create temporary lock file:", err)
	} else if err = lockfile.Close(); err != nil {
		t.Error("unable to close temporary lock file:", err)
	}
	defer os.Remove(lockfile.Name())

	// TODO: Add a test for cases where the file doesn't already exist. Need to
	// make sure this doesn't conflict with test files created in parallel.

	// Create a locker.
	locker, err := NewLocker(lockfile.Name(), 0600)
	if err != nil {
		t.Fatal("unable to create locker:", err)
	}

	// Attempt to acquire the lock.
	if err := locker.Lock(true); err != nil {
		t.Fatal("unable to acquire lock:", err)
	}

	// TODO: We should try to lock again to make sure that we can't, but this
	// behavior almost certainly isn't portable. We'd probably have to start a
	// new process to have a chance at testing this.

	// Attempt to release the lock.
	if err := locker.Unlock(); err != nil {
		t.Fatal("unable to release lock:", err)
	}
}
