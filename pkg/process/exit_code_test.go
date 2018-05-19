package process

import (
	"os/exec"
	"testing"

	"github.com/pkg/errors"
)

func TestExitCodeForNilError(t *testing.T) {
	if _, err := ExitCodeForError(nil); err == nil {
		t.Error("exit code was returned for nil error")
	}
}

func TestExitCodeForInvalidError(t *testing.T) {
	if _, err := ExitCodeForError(errors.New("not an exec error")); err == nil {
		t.Error("exit code was returned for invalid error")
	}
}

// TODO: It doesn't seem like there's anyway to test extraction of the
// syscall.WaitStatus from the error, because we can't construct an
// os.ProcessState (and it's not documented that we can rely on its zero value).
// Maybe look into this further?

func TestExitCode(t *testing.T) {
	// Run "go mutagen-test-invalid", which should return an error code of 2,
	// and verify its exit code.
	if err := exec.Command("go", "mutagen-test-invalid").Run(); err == nil {
		t.Fatal("expected non-nil error when running invalid Go command")
	} else if code, codeErr := ExitCodeForError(err); codeErr != nil {
		t.Fatal("unable to extract error exit code:", codeErr)
	} else if code != 2 {
		t.Error("exit code did not match expected")
	}
}
