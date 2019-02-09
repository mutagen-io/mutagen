package filesystem

import (
	"os"
	"runtime"
	"testing"
)

func TestDecomposesUnicodeByPathDarwinHFS(t *testing.T) {
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

	// Probe the behavior of the root and ensure it matches what's expected.
	if decomposes, err := DecomposesUnicodeByPath(hfsRoot); err != nil {
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

	// Probe the behavior of the root and ensure it matches what's expected.
	if decomposes, err := DecomposesUnicodeByPath(apfsRoot); err != nil {
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

	// Probe the behavior of the root and ensure it matches what's expected. The
	// only case we expect to decompose is HFS+ on Darwin, which we won't
	// encounter in this test.
	if decomposes, err := DecomposesUnicodeByPath("."); err != nil {
		t.Fatal("unable to probe Unicode decomposition:", err)
	} else if decomposes {
		t.Error("Unicode decomposition behavior does not match expected")
	}
}

// decomposesUnicodeTestCase represents a test case for DecomposesUnicode.
type decomposesUnicodeTestCase struct {
	// path is the path to test.
	path string
	// expected is the expected result of the Unicode decomposition test.
	expected bool
}

// run executes the test in the provided test context.
func (c *decomposesUnicodeTestCase) run(t *testing.T) {
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
	if decomposes, err := DecomposesUnicode(directory); err != nil {
		t.Fatal("unable to probe Unicode decomposition:", err)
	} else if decomposes != c.expected {
		t.Error("Unicode decomposition behavior does not match expected")
	}
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

	// Create the test case.
	testCase := &decomposesUnicodeTestCase{
		path:     HomeDirectory,
		expected: false,
	}

	// Run the test case.
	testCase.run(t)
}
