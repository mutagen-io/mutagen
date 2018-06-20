package filesystem

import (
	"os"
	"runtime"
	"testing"
)

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

	// Probe the behavior of the root and ensure it matches what's expected.
	if decomposes, err := DecomposesUnicode(hfsRoot); err != nil {
		t.Fatal("unable to probe Unicode decomposition:", err)
	} else if !decomposes {
		t.Error("Unicode decomposition behavior does not match expected")
	}
}

func TestDecomposesUnicodeDarwinAPFS(t *testing.T) {
	// If we don't have the separate APFS partition, skip this test.
	apfsRoot := os.Getenv("MUTAGEN_TEST_APFS_ROOT")
	if apfsRoot == "" {
		t.Skip()
	}

	// Probe the behavior of the root and ensure it matches what's expected.
	if decomposes, err := DecomposesUnicode(apfsRoot); err != nil {
		t.Fatal("unable to probe Unicode decomposition:", err)
	} else if decomposes {
		t.Error("Unicode decomposition behavior does not match expected")
	}
}

func TestDecomposesUnicodeOSPartition(t *testing.T) {
	// If we're on Darwin, then our OS partition could be either HFS+ (or some
	// variant thereof) or APFS, but it's difficult to know, so skip this test
	// in that case.
	if runtime.GOOS == "darwin" {
		t.Skip()
	}

	// Probe the behavior of the root and ensure it matches what's expected. The
	// only case we expect to decompose is HFS+ on Darwin, which we won't
	// encounter in this test.
	if decomposes, err := DecomposesUnicode("."); err != nil {
		t.Fatal("unable to probe Unicode decomposition:", err)
	} else if decomposes {
		t.Error("Unicode decomposition behavior does not match expected")
	}
}
