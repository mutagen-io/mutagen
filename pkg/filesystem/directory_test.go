package filesystem

import (
	"io/ioutil"
	"os"
	"runtime"
	"testing"
)

func TestDirectoryContentsNotExist(t *testing.T) {
	if _, err := DirectoryContents("/does/not/exist"); err == nil {
		t.Error("directory listing succeedeed for non-existent path")
	}
}

func TestDirectoryContentsFile(t *testing.T) {
	// Create an empty temporary file and defer its cleanup.
	file, err := ioutil.TempFile("", "mutagen_filesystem")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Error("unable to close temporary file:", err)
	}
	defer os.Remove(file.Name())

	// Ensure that directory listing fails.
	if _, err := DirectoryContents(file.Name()); err == nil {
		t.Error("directory listing succeedeed for non-directory path")
	}
}

func TestDirectoryContentsGOROOT(t *testing.T) {
	if contents, err := DirectoryContents(runtime.GOROOT()); err != nil {
		t.Fatal("directory listing failed for GOROOT:", err)
	} else if contents == nil {
		t.Fatal("directory contents nil for GOROOT")
	}
}

// TODO: If on Darwin, we should create and mount a virtual HFS+ volume to test
// normalization.
