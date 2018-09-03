package daemon

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/havoc-io/mutagen/pkg/mutagen"
)

const (
	// lockTestExecutablePackage is the Go package to build for running
	// concurrent lock tests.
	lockTestExecutablePackage = "github.com/havoc-io/mutagen/pkg/daemon/locktest"

	// lockTestFailMessage is a sentinel message used to indicate lock
	// acquisition failure in the test executable. We could use an exit code,
	// but "go run" doesn't forward them and different systems might handle them
	// differently.
	lockTestFailMessage = "Mutagen lock acquisition failed"
)

// TestLockCycle tests an acquisition/release cycle of the daemon lock.
func TestLockCycle(t *testing.T) {
	// Attempt to acquire the daemon lock.
	lock, err := AcquireLock()
	if err != nil {
		t.Fatal("unable to acquire lock:", err)
	}

	// Release the lock.
	if err := lock.Unlock(); err != nil {
		t.Fatal("unable to release lock:", err)
	}
}

// TestLockDuplicateFail tests that an additional attempt to acquire the daemon
// lock by a separate process will fail.
func TestLockDuplicateFail(t *testing.T) {
	// Compute the path to the Mutagen source tree.
	mutagenSourcePath, err := mutagen.SourceTreePath()
	if err != nil {
		t.Fatal("unable to compute path to Mutagen source tree:", err)
	}

	// Acquire the daemon lock and defer its release.
	lock, err := AcquireLock()
	if err != nil {
		t.Fatal("unable to acquire lock:", err)
	}
	defer lock.Unlock()

	// Attempt to run the test executable and ensure that it fails with the
	// proper error code (indicating failed lock acquisition).
	testCommand := exec.Command("go", "run", lockTestExecutablePackage)
	testCommand.Dir = mutagenSourcePath
	errorBuffer := &bytes.Buffer{}
	testCommand.Stderr = errorBuffer
	if err := testCommand.Run(); err == nil {
		t.Error("test command succeeded unexpectedly")
	} else if !strings.Contains(errorBuffer.String(), lockTestFailMessage) {
		t.Error("test command error output did not contain failure message")
	}
}
