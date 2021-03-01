package locking

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/mutagen"
)

const (
	// lockerTestExecutablePackage is the Go package to build for running
	// concurrent lock tests.
	lockerTestExecutablePackage = "github.com/mutagen-io/mutagen/pkg/filesystem/locking/lockertest"

	// lockerTestFailMessage is a sentinel message used to indicate lock
	// acquisition failure in the test executable. We could use an exit code,
	// but "go run" doesn't forward them and different systems might handle them
	// differently.
	lockerTestFailMessage = "lock acquisition failed"
)

// TestLockerFailOnDirectory tests that a locker creation fails for a directory.
func TestLockerFailOnDirectory(t *testing.T) {
	if _, err := NewLocker(t.TempDir(), 0600); err == nil {
		t.Fatal("creating a locker on a directory path succeeded")
	}
}

// TestLockerCycle tests the lifecycle of a Locker.
func TestLockerCycle(t *testing.T) {
	// Create a temporary file and defer its removal.
	lockfile, err := os.CreateTemp("", "mutagen_filesystem_lock")
	if err != nil {
		t.Fatal("unable to create temporary lock file:", err)
	} else if err = lockfile.Close(); err != nil {
		t.Error("unable to close temporary lock file:", err)
	}
	defer os.Remove(lockfile.Name())

	// Create a locker.
	locker, err := NewLocker(lockfile.Name(), 0600)
	if err != nil {
		t.Fatal("unable to create locker:", err)
	}

	// Attempt to acquire the lock.
	if err := locker.Lock(true); err != nil {
		t.Fatal("unable to acquire lock:", err)
	}

	// Verify that the lock state is correct.
	if !locker.Held() {
		t.Error("lock incorrectly reported as unlocked")
	}

	// Attempt to release the lock.
	if err := locker.Unlock(); err != nil {
		t.Fatal("unable to release lock:", err)
	}

	// Attempt to close the locker.
	if err := locker.Close(); err != nil {
		t.Fatal("unable to close locker:", err)
	}
}

// TestLockDuplicateFail tests that an additional attempt to acquire a lock by a
// separate process will fail.
func TestLockDuplicateFail(t *testing.T) {
	// Compute the path to the Mutagen source tree.
	mutagenSourcePath, err := mutagen.SourceTreePath()
	if err != nil {
		t.Fatal("unable to compute path to Mutagen source tree:", err)
	}

	// Create a temporary file and defer its removal.
	lockfile, err := os.CreateTemp("", "mutagen_filesystem_lock")
	if err != nil {
		t.Fatal("unable to create temporary lock file:", err)
	} else if err = lockfile.Close(); err != nil {
		t.Error("unable to close temporary lock file:", err)
	}
	defer os.Remove(lockfile.Name())

	// Create a locker for the file, acquire the lock, and defer the release of
	// the lock and closure of the locker.
	locker, err := NewLocker(lockfile.Name(), 0600)
	if err != nil {
		t.Fatal("unable to create locker:", err)
	} else if err = locker.Lock(true); err != nil {
		t.Fatal("unable to acquire lock:", err)
	}
	defer func() {
		locker.Unlock()
		locker.Close()
	}()

	// Attempt to run the test executable and ensure that it fails with the
	// proper error code (indicating failed lock acquisition).
	testCommand := exec.Command("go", "run", lockerTestExecutablePackage, lockfile.Name())
	testCommand.Dir = mutagenSourcePath
	errorBuffer := &bytes.Buffer{}
	testCommand.Stderr = errorBuffer
	if err := testCommand.Run(); err == nil {
		t.Error("test command succeeded unexpectedly")
	} else if !strings.Contains(errorBuffer.String(), lockerTestFailMessage) {
		t.Error("test command error output did not contain failure message", errorBuffer.String())
	}
}
