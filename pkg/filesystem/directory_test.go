package filesystem

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unicode/utf8"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
)

// TestPathSeparatorSingleByte verifies that the platform path separator rune is
// encoded as a single byte in UTF-8. We rely on this assumption for high
// performance in ensureValidName.
func TestPathSeparatorSingleByte(t *testing.T) {
	if utf8.RuneLen(os.PathSeparator) != 1 {
		t.Fatal("OS path separator does not have single-byte UTF-8 encoding")
	}
}

func TestDirectoryContentsNotExist(t *testing.T) {
	if _, err := DirectoryContentsByPath("/does/not/exist"); err == nil {
		t.Error("directory listing succeeded for non-existent path")
	}
}

func TestDirectoryContentsFile(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Create an empty temporary file and defer its cleanup.
	file, err := os.CreateTemp("", "mutagen_filesystem")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Error("unable to close temporary file:", err)
	}
	defer must.OSRemove(file.Name(), logger)

	// Ensure that directory listing fails.
	if _, err := DirectoryContentsByPath(file.Name()); err == nil {
		t.Error("directory listing succeeded for non-directory path")
	}
}

func TestDirectoryContentsGOROOT(t *testing.T) {
	if contents, err := DirectoryContentsByPath(runtime.GOROOT()); err != nil {
		t.Fatal("directory listing failed for GOROOT:", err)
	} else if contents == nil {
		t.Fatal("directory contents nil for GOROOT")
	}
}

// TestNonEmptyDirectoryRemovalFailure tests that removal of a non-empty
// directory results in failure.
func TestNonEmptyDirectoryRemovalFailure(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Create a handle for a temporary directory (that will be removed
	// automatically) and defer its closure.
	directory, _, err := OpenDirectory(t.TempDir(), false, logger)
	if err != nil {
		t.Fatal("unable to open directory handle:", err)
	}
	defer must.Close(directory, logger)

	// Create a directory that will serve as our target.
	if err := directory.CreateDirectory("target"); err != nil {
		t.Fatal("unable to create target directory:", err)
	}

	// Create content inside the directory.
	if target, err := directory.OpenDirectory("target", logger); err != nil {
		t.Fatal("unable to open target directory:", err)
	} else if err = target.CreateDirectory("content"); err != nil {
		must.Close(target, logger)
		t.Fatal("unable to create content in target directory:", err)
	} else if err = target.Close(); err != nil {
		t.Fatal("unable to close target directory:", err)
	}

	// Attempt to remove the target directory.
	if directory.RemoveDirectory("target") == nil {
		t.Error("able to remove non-empty directory")
	}
}

// TestDirectorySymbolicLinkRemoval tests that removal of symbolic links that
// point to directories works as expected.
func TestDirectorySymbolicLinkRemoval(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Create a temporary directory (that will be automatically removed).
	temporaryDirectoryPath := t.TempDir()

	// Create a handle for the temporary directory and defer its closure.
	directory, _, err := OpenDirectory(temporaryDirectoryPath, false, logger)
	if err != nil {
		t.Fatal("unable to open directory handle:", err)
	}
	defer must.Close(directory, logger)

	// Create a directory that will serve as our target.
	if err := directory.CreateDirectory("target"); err != nil {
		t.Fatal("unable to create target directory:", err)
	}

	// Open the target directory and defer its closure.
	target, err := directory.OpenDirectory("target", logger)
	if err != nil {
		t.Fatal("unable to open target directory:", err)
	}
	defer must.Close(target, logger)

	// Create content within the target.
	if err := target.CreateDirectory("content"); err != nil {
		t.Fatal("unable to create content in target directory:", err)
	}

	// Create a symbolic link to the target.
	if err := directory.CreateSymbolicLink("link", "target"); err != nil {
		t.Fatal("unable to create symbolic link:", err)
	}

	// Remove the symbolic link.
	if err := directory.RemoveSymbolicLink("link"); err != nil {
		t.Fatal("unable to remove symbolic link:", err)
	}

	// Grab the target metadata to ensure that it still exists. As an additional
	// sanity check, also ensure that the content path still exists, but do so
	// using path-based access.
	if _, err := directory.ReadContentMetadata("target"); err != nil {
		t.Error("unable to read target metadata:", err)
	} else if _, err = os.Lstat(filepath.Join(temporaryDirectoryPath, "target", "content")); err != nil {
		t.Error("unable to read target content metadata:", err)
	}
}
