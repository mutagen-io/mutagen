package agent

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pkg/errors"
)

func init() {
	// Compute the path to the test executable and its parent directory.
	executablePath, err := os.Executable()
	if err != nil {
		panic(errors.Wrap(err, "unable to compute test executable path"))
	}
	testDirectory := filepath.Dir(executablePath)

	// Compute the path to the agent bundle in the $GOPATH/bin directory.
	var agentBundlePath string
	if gopath := os.Getenv("GOPATH"); gopath == "" {
		panic(errors.New("GOPATH not set"))
	} else {
		agentBundlePath = filepath.Join(gopath, "bin", agentBundleName)
	}

	// Create a file that will be a copy of the agent bundle.
	// HACK: We're assuming that Go runs test executables inside temporary
	// directories that it cleans up, which does seem to be the case, but it'd
	// be nice if there were some way to remove the agent bundle ourselves,
	// maybe with some sort of atexit-like function.
	bundleCopyFile, err := os.Create(filepath.Join(testDirectory, agentBundleName))
	if err != nil {
		panic(errors.Wrap(err, "unable to create agent bundle copy file"))
	}
	defer bundleCopyFile.Close()

	// Open the agent bundle.
	bundleFile, err := os.Open(agentBundlePath)
	if err != nil {
		panic(errors.Wrap(err, "unable to open agent bundle file"))
	}
	defer bundleFile.Close()

	// Copy agent bundle contents.
	if _, err := io.Copy(bundleCopyFile, bundleFile); err != nil {
		panic(errors.Wrap(err, "unable to copy bundle file contents"))
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
