package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestExecutableForInvalidOS tests that ExecutableForPlatform fails for an
// invalid OS specification.
func TestExecutableForInvalidOS(t *testing.T) {
	if _, err := ExecutableForPlatform("fakeos", runtime.GOARCH, ""); err == nil {
		t.Fatal("extracting agent executable succeeded for invalid OS")
	}
}

// TestExecutableForInvalidArchitecture tests that ExecutableForPlatform fails
// for an invalid architecture specification.
func TestExecutableForInvalidArchitecture(t *testing.T) {
	if _, err := ExecutableForPlatform(runtime.GOOS, "fakearch", ""); err == nil {
		t.Fatal("extracting agent executable succeeded for invalid architecture")
	}
}

// TestExecutableForInvalidPair tests that ExecutableForPlatform fails for an
// invalid OS/architecture specification.
func TestExecutableForInvalidPair(t *testing.T) {
	if _, err := ExecutableForPlatform("fakeos", "fakearch", ""); err == nil {
		t.Fatal("extracting agent executable succeeded for invalid architecture")
	}
}

// TestExecutableForPlatform tests that ExecutableForPlatform succeeds for the
// current OS/architecture.
func TestExecutableForPlatform(t *testing.T) {
	if executable, err := ExecutableForPlatform(runtime.GOOS, runtime.GOARCH, ""); err != nil {
		t.Fatal("unable to extract agent bundle for current platform:", err)
	} else if err = os.Remove(executable); err != nil {
		t.Error("unable to remove agent executable:", err)
	}
}

// TestExecutableForPlatformWithOutputPath tests that ExecutableForPlatform
// functions correctly when an output path is specified.
func TestExecutableForPlatformWithOutputPath(t *testing.T) {
	// Compute the output path.
	outputPath := filepath.Join(t.TempDir(), "agent_output")

	// Perform executable extraction.
	executable, err := ExecutableForPlatform(runtime.GOOS, runtime.GOARCH, outputPath)
	if err != nil {
		t.Fatal("unable to extract agent bundle for current platform:", err)
	}

	// Verify the output path.
	if executable != outputPath {
		t.Error("executable output path does not match expected:", executable, "!=", outputPath)
	}

	// Remove the file.
	if err = os.Remove(executable); err != nil {
		t.Error("unable to remove agent executable:", err)
	}
}
