package filesystem

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/havoc-io/mutagen/pkg/mutagen"
)

const (
	// lockTestExecutablePackage is the Go package to build for running
	// concurrent lock tests.
	lockTestExecutablePackage = "github.com/havoc-io/mutagen/pkg/filesystem/locktest"

	// lockTestFailMessage is a sentinel message used to indicate lock
	// acquisition failure in the test executable. We could use an exit code,
	// but "go run" doesn't forward them and different systems might handle them
	// differently.
	lockTestFailMessage = "Mutagen lock acquisition failed"

	// testingDirectoryName is the name of a testing directory to create within
	// the Mutagen data directory.
	testingDirectoryName = "testing"
)

// TestMutagenLockCycle tests an acquisition/release cycle of the Mutagen lock.
func TestMutagenLockCycle(t *testing.T) {
	// Attempt to acquire the Mutagen lock.
	locker, err := AcquireMutagenLock()
	if err != nil {
		t.Fatal("unable to acquire lock:", err)
	}

	// Release the lock.
	if err := locker.Close(); err != nil {
		t.Fatal("unable to release lock:", err)
	}
}

// TestMutagenLockDuplicateFail tests that an additional attempt to acquire the
// Mutagen lock by a separate process will fail.
func TestLockDuplicateFail(t *testing.T) {
	// Compute the path to the Mutagen source tree.
	mutagenSourcePath, err := mutagen.SourceTreePath()
	if err != nil {
		t.Fatal("unable to compute path to Mutagen source tree:", err)
	}

	// Acquire the Mutagen lock and defer its release.
	locker, err := AcquireMutagenLock()
	if err != nil {
		t.Fatal("unable to acquire lock:", err)
	}
	defer locker.Close()

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

// TestMutagen tests the Mutagen data directory creation function.
func TestMutagen(t *testing.T) {
	// Attempt to create the testing subdirectory and defer its removal.
	path, err := Mutagen(true, testingDirectoryName)
	if err != nil {
		t.Fatal("unable to create testing subdirectory:", err)
	}
	defer os.RemoveAll(path)

	// Ensure it exists and is a directory.
	if info, err := os.Lstat(path); err != nil {
		t.Fatal("unable to probe testing subdirectory:", err)
	} else if !info.IsDir() {
		t.Error("Mutagen subpath is not a directory")
	}
}
