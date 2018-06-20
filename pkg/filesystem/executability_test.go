package filesystem

import (
	"os"
	"runtime"
	"testing"
)

func TestPreservesExecutabilityOSPartition(t *testing.T) {
	// Determine whether or not we expect the OS partition to preserve
	// executability.
	expected := runtime.GOOS != "windows"

	// Probe the current directory.
	if preserves, err := PreservesExecutability("."); err != nil {
		t.Fatal("unable to probe executability preservation:", err)
	} else if preserves != expected {
		t.Error("executability preservation behavior does not match expected")
	}
}

func TestPreservesExecutabilityFAT32(t *testing.T) {
	// If we don't have the separate FAT32 partition, skip this test.
	fat32Root := os.Getenv("MUTAGEN_TEST_FAT32_ROOT")
	if fat32Root == "" {
		t.Skip()
	}

	// Probe the FAT32 partition.
	if preserves, err := PreservesExecutability(fat32Root); err != nil {
		t.Fatal("unable to probe executability preservation:", err)
	} else if preserves {
		t.Error("executability preservation detected on FAT32 partition")
	}
}
