package behavior

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
)

func TestDecomposesUnicodeByPathAssumedHomeDirectory(t *testing.T) {
	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("unable to compute home directory:", err)
	}

	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Query the assumed behavior of the home directory and ensure it matches
	// what's expected.
	//
	// TODO: We should perform some validation on the second parameter returned
	// by DecomposesUnicodeByPath (indicating whether or not probe files were
	// used).
	if decomposes, _, err := DecomposesUnicodeByPath(homeDirectory, ProbeMode_ProbeModeAssume, logger); err != nil {
		t.Fatal("unable to query Unicode decomposition:", err)
	} else if decomposes {
		t.Error("Unicode decomposition behavior does not match expected")
	}
}

func TestDecomposesUnicodeByPathDarwinHFS(t *testing.T) {
	// If we're not on Darwin, skip this test. We may have an HFS+ root (e.g. on
	// Linux), but Linux's HFS+ implementation can either compose or decompose
	// depending on its settings.
	if runtime.GOOS != "darwin" {
		t.Skip()
	}
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// If we don't have the separate HFS+ partition, skip this test.
	hfsRoot := os.Getenv("MUTAGEN_TEST_HFS_ROOT")
	if hfsRoot == "" {
		t.Skip()
	}

	// Probe the behavior of the root and ensure it matches what's expected.
	//
	// TODO: We should perform some validation on the second parameter returned
	// by DecomposesUnicodeByPath (indicating whether or not probe files were
	// used).
	if decomposes, _, err := DecomposesUnicodeByPath(hfsRoot, ProbeMode_ProbeModeProbe, logger); err != nil {
		t.Fatal("unable to probe Unicode decomposition:", err)
	} else if !decomposes {
		t.Error("Unicode decomposition behavior does not match expected")
	}
}

func TestDecomposesUnicodeByPathDarwinAPFS(t *testing.T) {
	// If we don't have the separate APFS partition, skip this test.
	apfsRoot := os.Getenv("MUTAGEN_TEST_APFS_ROOT")
	if apfsRoot == "" {
		t.Skip()
	}
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Probe the behavior of the root and ensure it matches what's expected.
	//
	// TODO: We should perform some validation on the second parameter returned
	// by DecomposesUnicodeByPath (indicating whether or not probe files were
	// used).
	if decomposes, _, err := DecomposesUnicodeByPath(apfsRoot, ProbeMode_ProbeModeProbe, logger); err != nil {
		t.Fatal("unable to probe Unicode decomposition:", err)
	} else if decomposes {
		t.Error("Unicode decomposition behavior does not match expected")
	}
}

func TestDecomposesUnicodeByPathOSPartition(t *testing.T) {
	// If we're on Darwin, then our OS partition could be either HFS+ (or some
	// variant thereof) or APFS, but it's difficult to know, so skip this test
	// in that case.
	if runtime.GOOS == "darwin" {
		t.Skip()
	}

	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Probe the behavior of the root and ensure it matches what's expected. The
	// only case we expect to decompose is HFS+ on Darwin, which we won't
	// encounter in this test.
	//
	// TODO: We should perform some validation on the second parameter returned
	// by DecomposesUnicodeByPath (indicating whether or not probe files were
	// used).
	if decomposes, _, err := DecomposesUnicodeByPath(".", ProbeMode_ProbeModeProbe, logger); err != nil {
		t.Fatal("unable to probe Unicode decomposition:", err)
	} else if decomposes {
		t.Error("Unicode decomposition behavior does not match expected")
	}
}

// decomposesUnicodeTestCase represents a test case for DecomposesUnicode.
type decomposesUnicodeTestCase struct {
	// path is the path to test.
	path string
	// assume indicates that an assumption should be generated as opposed to
	// actual probing.
	assume bool
	// expected is the expected result of the Unicode decomposition test.
	expected bool

	logger *logging.Logger
}

// run executes the test in the provided test context.
func (c *decomposesUnicodeTestCase) run(t *testing.T) {
	// Mark ourselves as a helper function.
	t.Helper()

	// Open the path, ensure that it's a directory, and defer its closure.
	directory, _, err := filesystem.OpenDirectory(c.path, false, c.logger)
	if err != nil {
		t.Fatal("unable to open path:", err)
	}
	defer must.Close(directory, c.logger)

	// Determine the probing mode.
	probeMode := ProbeMode_ProbeModeProbe
	if c.assume {
		probeMode = ProbeMode_ProbeModeAssume
	}

	// Probe the behavior of the root and ensure it matches what's expected.
	//
	// TODO: We should perform some validation on the second parameter returned
	// by DecomposesUnicode (indicating whether or not probe files were used).
	if decomposes, _, err := DecomposesUnicode(directory, probeMode, c.logger); err != nil {
		t.Fatal("unable to probe Unicode decomposition:", err)
	} else if decomposes != c.expected {
		t.Error("Unicode decomposition behavior does not match expected")
	}
}

// TestDecomposesUnicodeAssumedHomeDirectory tests assumed Unicode decomposition
// behavior on the home directory.
func TestDecomposesUnicodeAssumedHomeDirectory(t *testing.T) {
	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("unable to compute home directory:", err)
	}

	// Create the test case.
	testCase := &decomposesUnicodeTestCase{
		path:     homeDirectory,
		assume:   true,
		expected: false,
		logger:   logging.NewLogger(logging.LevelError, &bytes.Buffer{}),
	}

	// Run the test case.
	testCase.run(t)
}

// TestDecomposesUnicodeDarwinHFS tests Unicode decomposition behavior on an
// HFS+ partition on Darwin, if available.
func TestDecomposesUnicodeDarwinHFS(t *testing.T) {
	// If we're not on Darwin, skip this test. We may have an HFS+ root (e.g. on
	// Linux), but Linux's HFS+ implementation can either compose or decompose
	// depending on its settings.
	if runtime.GOOS != "darwin" {
		t.Skip()
	}

	// If we don't have the separate HFS+ partition, skip this test.
	hfsRoot := os.Getenv("MUTAGEN_TEST_HFS_ROOT")
	if hfsRoot == "" {
		t.Skip()
	}

	// Create the test case.
	testCase := &decomposesUnicodeTestCase{
		path:     hfsRoot,
		expected: true,
		logger:   logging.NewLogger(logging.LevelError, &bytes.Buffer{}),
	}

	// Run the test case.
	testCase.run(t)
}

// TestDecomposesUnicodeDarwinAPFS tests Unicode decomposition behavior on an
// APFS partition on Darwin, if available.
func TestDecomposesUnicodeDarwinAPFS(t *testing.T) {
	// If we don't have the separate APFS partition, skip this test.
	apfsRoot := os.Getenv("MUTAGEN_TEST_APFS_ROOT")
	if apfsRoot == "" {
		t.Skip()
	}

	// Create the test case.
	testCase := &decomposesUnicodeTestCase{
		path:     apfsRoot,
		expected: false,
		logger:   logging.NewLogger(logging.LevelError, &bytes.Buffer{}),
	}

	// Run the test case.
	testCase.run(t)
}

// TestDecomposesUnicodeHomeDirectory tests Unicode decomposition behavior on
// the home directory on non-Darwin systems.
func TestDecomposesUnicodeHomeDirectory(t *testing.T) {
	// If we're on Darwin, then our OS partition could be either HFS+ (or some
	// variant thereof) or APFS, but it's difficult to know, so skip this test
	// in that case.
	if runtime.GOOS == "darwin" {
		t.Skip()
	}

	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("unable to compute home directory:", err)
	}

	// Create the test case.
	testCase := &decomposesUnicodeTestCase{
		path:     homeDirectory,
		expected: false,
		logger:   logging.NewLogger(logging.LevelError, &bytes.Buffer{}),
	}

	// Run the test case.
	testCase.run(t)
}
