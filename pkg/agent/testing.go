package agent

// HACK: This file is necessary to export functions that are only used in
// testing but which need to be imported by other packages to run their tests.
// Ideally we could place these in _test.go files and import them in other
// packages, but Go does not allow this (it only looks at _test.go files in the
// package being tested, so it won't see these functions if we put them in a
// _test.go file in a package that we're importing).

import (
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/mutagen"
)

const (
	// buildDirectoryName is the name of the Mutagen build directory at the root
	// of the Mutagen source tree.
	buildDirectoryName = "build"
)

// CopyBundleForTesting copies the agent bundle from it's build path alongside
// the current executable. It is useful for copying the agent bundle next to the
// current test executable.
func CopyBundleForTesting() error {
	// Compute the path to the test executable and its parent directory.
	executablePath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "unable to compute test executable path")
	}
	testDirectory := filepath.Dir(executablePath)

	// Compute the path to the Mutagen source directory.
	mutagenSourcePath, err := mutagen.SourceTreePath()
	if err != nil {
		return errors.Wrap(err, "unable to compute Mutagen source tree path")
	}

	// Compute the path to the agent bundle in the build directory.
	agentBundlePath := filepath.Join(mutagenSourcePath, buildDirectoryName, BundleName)

	// Create a file that will be a copy of the agent bundle.
	// HACK: We're assuming that Go runs test executables inside temporary
	// directories that it cleans up, which does seem to be the case, but it'd
	// be nice if there were some way to remove the agent bundle ourselves,
	// maybe with some sort of atexit-like function.
	bundleCopyFile, err := os.Create(filepath.Join(testDirectory, BundleName))
	if err != nil {
		return errors.Wrap(err, "unable to create agent bundle copy file")
	}
	defer bundleCopyFile.Close()

	// Open the agent bundle.
	bundleFile, err := os.Open(agentBundlePath)
	if err != nil {
		return errors.Wrap(err, "unable to open agent bundle file")
	}
	defer bundleFile.Close()

	// Copy agent bundle contents.
	if _, err := io.Copy(bundleCopyFile, bundleFile); err != nil {
		return errors.Wrap(err, "unable to copy bundle file contents")
	}

	// Success.
	return nil
}
