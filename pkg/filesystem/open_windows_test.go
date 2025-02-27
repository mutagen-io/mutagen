package filesystem

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/logging"
)

// TestOpenLongPath verifies that calling Open succeeds on a directory whose
// path length exceeds the default path length limit on Windows.
func TestOpenLongPath(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})
	// Create a directory (in a temporary directory that will be automatically
	// removed) with a name that will exceed the Windows path length limit.
	longDirectoryName := strings.Repeat("d", windowsLongPathTestingLength)
	longtemporaryDirectoryPath := filepath.Join(t.TempDir(), longDirectoryName)
	if err := os.Mkdir(longtemporaryDirectoryPath, 0700); err != nil {
		t.Fatal("unable to create test directory with long name:", err)
	}

	// Attempt to open the directory and ensure doing so succeeds.
	directory, _, err := Open(longtemporaryDirectoryPath, false, logger)
	if err != nil {
		t.Fatal("unable to open directory with long path:", err)
	}
	directory.Close()
}
