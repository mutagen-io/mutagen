package filesystem

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOpenLongPath verifies that calling Open succeeds on a directory whose
// path length exceeds the default path length limit on Windows.
func TestOpenLongPath(t *testing.T) {
	// Create a temporary directory and defer its cleanup.
	temporaryDirectoryPath, err := ioutil.TempDir("", "parent")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(temporaryDirectoryPath)

	// Create a directory in the temporary directory with a name that will
	// exceed the Windows path length limit.
	longDirectoryName := strings.Repeat("d", windowsLongPathTestingLength)
	longtemporaryDirectoryPath := filepath.Join(temporaryDirectoryPath, longDirectoryName)
	if err := os.Mkdir(longtemporaryDirectoryPath, 0700); err != nil {
		t.Fatal("unable to create test directory with long name:", err)
	}

	// Attempt to open the directory and ensure doing so succeeds.
	directory, _, err := Open(longtemporaryDirectoryPath, false)
	if err != nil {
		t.Fatal("unable to open directory with long path:", err)
	}
	directory.Close()
}
