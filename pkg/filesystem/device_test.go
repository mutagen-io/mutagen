package filesystem

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDeviceIDsDifferent(t *testing.T) {
	// If we're on Windows, the device ID is always 0, so skip this test in that
	// case.
	if runtime.GOOS == "windows" {
		t.Skip()
	}

	// If we don't have the separate FAT32 partition, skip this test.
	fat32Root := os.Getenv("MUTAGEN_TEST_FAT32_ROOT")
	if fat32Root == "" {
		t.Skip()
	}

	// Grab the device ID for the current path.
	info, err := os.Lstat(".")
	if err != nil {
		t.Fatal("lstat failed for current path:", err)
	}
	deviceID, err := DeviceID(info)
	if err != nil {
		t.Fatal("device ID probe failed for current path:", err)
	}

	// Grab the device ID for the FAT32 partition.
	fat32Info, err := os.Lstat(fat32Root)
	if err != nil {
		t.Fatal("lstat failed for FAT32 partition:", err)
	}
	fat32DeviceID, err := DeviceID(fat32Info)
	if err != nil {
		t.Fatal("device ID probe failed for FAT32 partition:", err)
	}

	// Ensure they differ.
	if deviceID == fat32DeviceID {
		t.Error("different partitions show same device ID")
	}
}

func TestDeviceIDSubrootDifferent(t *testing.T) {
	// If we're on Windows, the device ID is always 0, so skip this test in that
	// case.
	if runtime.GOOS == "windows" {
		t.Skip()
	}

	// If we don't have the separate FAT32 partition mounted at a subdirectory,
	// skip this test.
	fat32Subroot := os.Getenv("MUTAGEN_TEST_FAT32_SUBROOT")
	if fat32Subroot == "" {
		t.Skip()
	}

	// Compute its parent path.
	parent := filepath.Dir(fat32Subroot)

	// Grab the device ID for the parent path.
	parentInfo, err := os.Lstat(parent)
	if err != nil {
		t.Fatal("lstat failed for parent path:", err)
	}
	parentDeviceID, err := DeviceID(parentInfo)
	if err != nil {
		t.Fatal("device ID probe failed for parent path:", err)
	}

	// Grab the device ID for the FAT32 partition.
	fat32SubrootInfo, err := os.Lstat(fat32Subroot)
	if err != nil {
		t.Fatal("lstat failed for FAT32 subpath:", err)
	}
	fat32SubrootDeviceID, err := DeviceID(fat32SubrootInfo)
	if err != nil {
		t.Fatal("device ID probe failed for FAT32 subpath:", err)
	}

	// Ensure they differ.
	if fat32SubrootDeviceID == parentDeviceID {
		t.Error("separate partition has same device ID as parent path")
	}
}
