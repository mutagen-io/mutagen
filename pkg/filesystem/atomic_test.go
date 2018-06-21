package filesystem

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomicNonExistentDirectory(t *testing.T) {
	if WriteFileAtomic("/does/not/exist", []byte{}, 0600) == nil {
		t.Error("atomic file write did not fail for non-existent path")
	}
}

func TestWriteFileAtomic(t *testing.T) {
	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_write_file_atomic")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Compute the target path.
	target := filepath.Join(directory, "file")

	// Create contents.
	contents := []byte{0, 1, 2, 3, 4, 5, 6}

	// Attempt to write to a temporary file.
	if err := WriteFileAtomic(target, contents, 0600); err != nil {
		t.Fatal("atomic file write failed:", err)
	}

	// Read the contents back and ensure they match what's expected.
	if data, err := ioutil.ReadFile(target); err != nil {
		t.Fatal("unable to read back file:", err)
	} else if !bytes.Equal(data, contents) {
		t.Error("file contents did not match expected")
	}
}

func TestRenameFileAtomicSameDevice(t *testing.T) {
	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_rename_file_atomic")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Compute source path.
	source := filepath.Join(directory, "source")

	// Create contents.
	contents := []byte{0, 1, 2, 3, 4, 5, 6}

	// Attempt to write to a temporary file.
	if err := WriteFileAtomic(source, contents, 0600); err != nil {
		t.Fatal("atomic file write failed:", err)
	}

	// Compute target path.
	target := filepath.Join(directory, "target")

	// Rename the file.
	if err := RenameFileAtomic(source, target); err != nil {
		t.Fatal("unable to rename file:", err)
	}

	// Read the contents back and ensure they match what's expected.
	if data, err := ioutil.ReadFile(target); err != nil {
		t.Fatal("unable to read back file:", err)
	} else if !bytes.Equal(data, contents) {
		t.Error("file contents did not match expected")
	}
}

func TestRenameFileAtomicDifferentDevice(t *testing.T) {
	// If we don't have the separate FAT32 partition, skip this test.
	fat32Root := os.Getenv("MUTAGEN_TEST_FAT32_ROOT")
	if fat32Root == "" {
		t.Skip()
	}

	// Create a temporary directory on the host partition and defer its cleanup.
	sourceDirectory, err := ioutil.TempDir("", "mutagen_rename_file_atomic")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(sourceDirectory)

	// Compute source path.
	source := filepath.Join(sourceDirectory, "source")

	// Create contents.
	contents := []byte{0, 1, 2, 3, 4, 5, 6}

	// Attempt to write to a temporary file.
	if err := WriteFileAtomic(source, contents, 0600); err != nil {
		t.Fatal("atomic file write failed:", err)
	}

	// Create a temporary directory on the FAT32 partition and defer its cleanup.
	targetDirectory, err := ioutil.TempDir(fat32Root, "mutagen_rename_file_atomic")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(targetDirectory)

	// Compute target path.
	target := filepath.Join(targetDirectory, "target")

	// Rename the file.
	if err := RenameFileAtomic(source, target); err != nil {
		t.Fatal("unable to rename file:", err)
	}

	// Read the contents back and ensure they match what's expected.
	if data, err := ioutil.ReadFile(target); err != nil {
		t.Fatal("unable to read back file:", err)
	} else if !bytes.Equal(data, contents) {
		t.Error("file contents did not match expected")
	}
}
