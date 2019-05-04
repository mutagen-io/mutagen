package agent

import (
	"os"
	"runtime"
	"testing"
)

// TestExecutableForInvalidOS tests that executableForPlatform fails for an
// invalid OS specification.
func TestExecutableForInvalidOS(t *testing.T) {
	if _, err := executableForPlatform("fakeos", runtime.GOARCH); err == nil {
		t.Fatal("extracting agent executable succeeded for invalid OS")
	}
}

// TestExecutableForInvalidArchitecture tests that executableForPlatform fails
// for an invalid architecture specification.
func TestExecutableForInvalidArchitecture(t *testing.T) {
	if _, err := executableForPlatform(runtime.GOOS, "fakearch"); err == nil {
		t.Fatal("extracting agent executable succeeded for invalid architecture")
	}
}

// TestExecutableForInvalidPair tests that executableForPlatform fails for an
// invalid OS/architecture specification.
func TestExecutableForInvalidPair(t *testing.T) {
	if _, err := executableForPlatform("fakeos", "fakearch"); err == nil {
		t.Fatal("extracting agent executable succeeded for invalid architecture")
	}
}

// TestExecutableForPlatform tests that executableForPlatform succeeds for the
// current OS/architecture.
func TestExecutableForPlatform(t *testing.T) {
	if executable, err := executableForPlatform(runtime.GOOS, runtime.GOARCH); err != nil {
		t.Fatal("unable to extract agent bundle for current platform:", err)
	} else if err = os.Remove(executable); err != nil {
		t.Error("unable to remove agent executable:", err)
	}
}
