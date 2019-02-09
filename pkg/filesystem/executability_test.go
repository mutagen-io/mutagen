package filesystem

import (
	"os"
	"runtime"
	"testing"
)

// preservesExecutabilityByPathTestCase represents a test case for
// PreservesExecutabilityByPath.
type preservesExecutabilityByPathTestCase struct {
	// path is the path to test.
	path string
	// expected is the expected result of the executability preservation test.
	expected bool
}

// run executes the test in the provided test context.
func (c *preservesExecutabilityByPathTestCase) run(t *testing.T) {
	// Mark ourselves as a helper function.
	t.Helper()

	// Probe the behavior of the root and ensure it matches what's expected.
	if preserves, err := PreservesExecutabilityByPath(c.path); err != nil {
		t.Fatal("unable to probe executability preservation:", err)
	} else if preserves != c.expected {
		t.Error("executability preservation behavior does not match expected")
	}
}

// TestPreservesExecutabilityByPathHomeDirectory tests executability
// preservation behavior by path on the home directory.
func TestPreservesExecutabilityByPathHomeDirectory(t *testing.T) {
	// Create the test case.
	testCase := &preservesExecutabilityByPathTestCase{
		path:     HomeDirectory,
		expected: runtime.GOOS != "windows",
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
	}

	// Run the test case.
	testCase.run(t)
}

// preservesExecutabilityTestCase represents a test case for
// PreservesExecutability.
type preservesExecutabilityTestCase struct {
	// path is the path to test.
	path string
	// expected is the expected result of the executability preservation test.
	expected bool
}

// run executes the test in the provided test context.
func (c *preservesExecutabilityTestCase) run(t *testing.T) {
	// Mark ourselves as a helper function.
	t.Helper()

	// Open the path, ensure that it's a directory, and defer its closure.
	object, metadata, err := Open(c.path, false)
	var directory *Directory
	var ok bool
	if err != nil {
		t.Fatal("unable to open path:", err)
	} else if metadata.Mode&ModeTypeMask != ModeTypeDirectory {
		t.Fatal("path is not a directory")
	} else if directory, ok = object.(*Directory); !ok {
		t.Fatal("filesystem object did not convert to directory")
	}
	defer directory.Close()

	// Probe the behavior of the root and ensure it matches what's expected.
	if preserves, err := PreservesExecutability(directory); err != nil {
		t.Fatal("unable to probe executability preservation:", err)
	} else if preserves != c.expected {
		t.Error("executability preservation behavior does not match expected")
	}
}

// TestPreservesExecutabilityHomeDirectory tests executability preservation
// behavior on the home directory.
func TestPreservesExecutabilityHomeDirectory(t *testing.T) {
	// Create the test case.
	testCase := &preservesExecutabilityTestCase{
		path:     HomeDirectory,
		expected: runtime.GOOS != "windows",
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
	}

	// Run the test case.
	testCase.run(t)
}
