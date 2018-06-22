package agent

import (
	"os"
	"runtime"
	"testing"

	"github.com/pkg/errors"
)

func init() {
	// Copy the agent bundle for testing.
	// HACK: We're relying on the fact that Go will clean this up when it
	// removes the testing temporary directory.
	if err := CopyBundleForTesting(); err != nil {
		panic(errors.Wrap(err, "unable to copy agent bundle for testing"))
	}
}

func TestExecutableForInvalidPlatform(t *testing.T) {
	if _, err := executableForPlatform("fakeos", "fakearch"); err == nil {
		t.Fatal("extracting agent executable succeeded for invalid platform")
	}
}

func TestExecutableForPlatform(t *testing.T) {
	if executable, err := executableForPlatform(runtime.GOOS, runtime.GOARCH); err != nil {
		t.Fatal("unable to extract agent bundle for current platform:", err)
	} else if err = os.Remove(executable); err != nil {
		t.Error("unable to remove agent executable:", err)
	}
}
