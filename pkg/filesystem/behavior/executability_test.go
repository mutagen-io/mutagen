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

// preservesExecutabilityByPathTestCase represents a test case for
// PreservesExecutabilityByPath.
type preservesExecutabilityByPathTestCase struct {
	// path is the path to test.
	path string
	// assume indicates that an assumption should be generated as opposed to
	// actual probing.
	assume bool
	// expected is the expected result of the executability preservation test.
	expected bool
	logger   *logging.Logger
}

// run executes the test in the provided test context.
func (c *preservesExecutabilityByPathTestCase) run(t *testing.T) {
	// Mark ourselves as a helper function.
	t.Helper()

	// Determine the probing mode.
	probeMode := ProbeMode_ProbeModeProbe
	if c.assume {
		probeMode = ProbeMode_ProbeModeAssume
	}

	// Probe the behavior of the root and ensure it matches what's expected.
	//
	// TODO: We should perform some validation on the second parameter returned
	// by PreservesExecutabilityByPath (indicating whether or not probe files
	// were used).
	if preserves, _, err := PreservesExecutabilityByPath(c.path, probeMode, c.logger); err != nil {
		t.Fatal("unable to probe executability preservation:", err)
	} else if preserves != c.expected {
		t.Error("executability preservation behavior does not match expected")
	}
}

// TestPreservesExecutabilityByPathAssumedHomeDirectory tests assumed
// executability preservation behavior by path on the current directory.
func TestPreservesExecutabilityByPathAssumedHomeDirectory(t *testing.T) {
	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("unable to compute home directory:", err)
	}

	// Create the test case.
	testCase := &preservesExecutabilityByPathTestCase{
		path:     homeDirectory,
		assume:   true,
		expected: runtime.GOOS != "windows",
		logger:   logging.NewLogger(logging.LevelError, &bytes.Buffer{}),
	}

	// Run the test case.
	testCase.run(t)
}

// TestPreservesExecutabilityByPathHomeDirectory tests executability
// preservation behavior by path on the home directory.
func TestPreservesExecutabilityByPathHomeDirectory(t *testing.T) {
	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("unable to compute home directory:", err)
	}

	// Create the test case.
	testCase := &preservesExecutabilityByPathTestCase{
		path:     homeDirectory,
		expected: runtime.GOOS != "windows",
		logger:   logging.NewLogger(logging.LevelError, &bytes.Buffer{}),
	}

	// Run the test case.
	testCase.run(t)
}

// TestPreservesExecutabilityByPathFAT32 tests executability preservation
// behavior by path on a FAT32 partition, if available.
func TestPreservesExecutabilityByPathFAT32(t *testing.T) {
	// If we don't have the separate FAT32 partition, skip this test.
	fat32Root := os.Getenv("MUTAGEN_TEST_FAT32_ROOT")
	if fat32Root == "" {
		t.Skip()
	}

	// Create the test case.
	testCase := &preservesExecutabilityByPathTestCase{
		path:     fat32Root,
		expected: false,
		logger:   logging.NewLogger(logging.LevelError, &bytes.Buffer{}),
	}

	// Run the test case.
	testCase.run(t)
}

// preservesExecutabilityTestCase represents a test case for
// PreservesExecutability.
type preservesExecutabilityTestCase struct {
	// path is the path to test.
	path string
	// assume indicates that an assumption should be generated as opposed to
	// actual probing.
	assume bool
	// expected is the expected result of the executability preservation test.
	expected bool

	logger *logging.Logger
}

// run executes the test in the provided test context.
func (c *preservesExecutabilityTestCase) run(t *testing.T) {
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
	// by PreservesExecutability (indicating whether or not probe files were
	// used).
	if preserves, _, err := PreservesExecutability(directory, probeMode, c.logger); err != nil {
		t.Fatal("unable to probe executability preservation:", err)
	} else if preserves != c.expected {
		t.Error("executability preservation behavior does not match expected")
	}
}

// TestPreservesExecutabilityAssumedHomeDirectory tests assumed executability
// preservation behavior on the home directory.
func TestPreservesExecutabilityAssumedHomeDirectory(t *testing.T) {
	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("unable to compute home directory:", err)
	}

	// Create the test case.
	testCase := &preservesExecutabilityTestCase{
		path:     homeDirectory,
		assume:   true,
		expected: runtime.GOOS != "windows",
		logger:   logging.NewLogger(logging.LevelError, &bytes.Buffer{}),
	}

	// Run the test case.
	testCase.run(t)
}

// TestPreservesExecutabilityHomeDirectory tests executability preservation
// behavior on the home directory.
func TestPreservesExecutabilityHomeDirectory(t *testing.T) {
	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("unable to compute home directory:", err)
	}

	// Create the test case.
	testCase := &preservesExecutabilityTestCase{
		path:     homeDirectory,
		expected: runtime.GOOS != "windows",
		logger:   logging.NewLogger(logging.LevelError, &bytes.Buffer{}),
	}

	// Run the test case.
	testCase.run(t)
}

// TestPreservesExecutabilityFAT32 tests executability preservation behavior on
// a FAT32 partition, if available.
func TestPreservesExecutabilityFAT32(t *testing.T) {
	// If we don't have the separate FAT32 partition, skip this test.
	fat32Root := os.Getenv("MUTAGEN_TEST_FAT32_ROOT")
	if fat32Root == "" {
		t.Skip()
	}

	// Create the test case.
	testCase := &preservesExecutabilityTestCase{
		path:     fat32Root,
		expected: false,
		logger:   logging.NewLogger(logging.LevelError, &bytes.Buffer{}),
	}

	// Run the test case.
	testCase.run(t)
}
