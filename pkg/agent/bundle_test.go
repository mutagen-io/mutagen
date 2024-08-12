package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/logging"
)

// TestExecutableForInvalidOS tests that ExecutableForPlatform fails for an
// invalid OS specification.
func TestExecutableForInvalidOS(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})
	if _, err := ExecutableForPlatform("fakeos", runtime.GOARCH, "", logger); err == nil {
		t.Fatal("extracting agent executable succeeded for invalid OS")
	}
}

// TestExecutableForInvalidArchitecture tests that ExecutableForPlatform fails
// for an invalid architecture specification.
func TestExecutableForInvalidArchitecture(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})
	if _, err := ExecutableForPlatform(runtime.GOOS, "fakearch", "", logger); err == nil {
		t.Fatal("extracting agent executable succeeded for invalid architecture")
	}
}

// TestExecutableForInvalidPair tests that ExecutableForPlatform fails for an
// invalid OS/architecture specification.
func TestExecutableForInvalidPair(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})
	if _, err := ExecutableForPlatform("fakeos", "fakearch", "", logger); err == nil {
		t.Fatal("extracting agent executable succeeded for invalid architecture")
	}
}

// TestExecutableForPlatform tests that ExecutableForPlatform succeeds for the
// current OS/architecture.
func TestExecutableForPlatform(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})
	if executable, err := ExecutableForPlatform(runtime.GOOS, runtime.GOARCH, "", logger); err != nil {
		t.Fatal("unable to extract agent bundle for current platform:", err)
	} else if err = os.Remove(executable); err != nil {
		t.Error("unable to remove agent executable:", err)
	}
}

// TestExecutableForPlatformWithOutputPath tests that ExecutableForPlatform
// functions correctly when an output path is specified.
func TestExecutableForPlatformWithOutputPath(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})
	// Compute the output path.
	outputPath := filepath.Join(t.TempDir(), "agent_output")

	// Perform executable extraction.
	executable, err := ExecutableForPlatform(runtime.GOOS, runtime.GOARCH, outputPath, logger)
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
