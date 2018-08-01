package daemon

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/havoc-io/mutagen/pkg/process"
)

const (
	// lockTestExecutablePackage is the Go package to build for running
	// concurrent lock tests.
	lockTestExecutablePackage = "github.com/havoc-io/mutagen/pkg/daemon/locktest"

	// lockTestFailExitCode is a sentinel exit code used to indicate lock
	// acquisition failure in the test executable.
	lockTestFailExitCode = 64
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

// TestLockDuplicate tests that an additional attempt to acquire the daemon lock
// by a separate process will fail.
func TestLockDuplicate(t *testing.T) {
	// Create a temporary directory in which to build the lock test executable
	// and defer its removal.
	buildDirectory, err := ioutil.TempDir("", "mutagen_daemon_lock_test")
	if err != nil {
		t.Fatal("unable to create temporary build directory:", err)
	}
	defer os.RemoveAll(buildDirectory)

	// Build the test executable.
	buildCommand := exec.Command("go", "build", lockTestExecutablePackage)
	buildCommand.Dir = buildDirectory
	if err := buildCommand.Run(); err != nil {
		t.Fatal("unable to build test command:", err)
	}

	// Acquire the daemon lock and defer its release.
	lock, err := AcquireLock()
	if err != nil {
		t.Fatal("unable to acquire lock:", err)
	}
	defer lock.Unlock()

	// Compute the full path to the test executable.
	executablePath := filepath.Join(
		buildDirectory,
		process.ExecutableName(path.Base(lockTestExecutablePackage), runtime.GOOS),
	)

	// Attempt to run the test executable and ensure that it fails with the
	// proper error code (indicating failed lock acquisition).
	testCommand := exec.Command(executablePath)
	if err := testCommand.Run(); err == nil {
		t.Error("test command succeeded unexpectedly")
	} else if code, codeErr := process.ExitCodeForError(err); codeErr != nil {
		t.Error("unable to extract exit code from error:", codeErr)
	} else if code != lockTestFailExitCode {
		t.Error("unexpected exit code from test process:", code)
	}
}
