package filesystem

import (
	"bytes"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/must"
)

// TestDirectoryLongPaths tests a variety of Directory operations on directory
// and file names that exceed the default Windows path length limit.
func TestDirectoryLongPaths(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Create a temporary directory (that will be automatically removed).
	temporaryDirectoryPath := t.TempDir()

	// Create a directory in the temporary directory with a name that will
	// exceed the Windows path length limit.
	longDirectoryName := strings.Repeat("d", windowsLongPathTestingLength)
	if err := os.Mkdir(filepath.Join(temporaryDirectoryPath, longDirectoryName), 0700); err != nil {
		t.Fatal("unable to create test directory with long name:", err)
	}

	// Create a file in the temporary directory with a name that will exceed the
	// Windows path length limit.
	longFileName := strings.Repeat("f", windowsLongPathTestingLength)
	file, err := os.Create(filepath.Join(temporaryDirectoryPath, longFileName))
	if err != nil {
		t.Fatal("unable to create test file with long name:", err)
	}
	must.Close(file, logger)

	// Open the temporary directory for access and defer its closure.
	closer, _, err := Open(temporaryDirectoryPath, false, logger)
	if err != nil {
		t.Fatal("unable to open directory:", err)
	}
	defer must.Close(closer, logger)

	// Extract the directory object.
	var directory *Directory
	if d, ok := closer.(*Directory); !ok {
		t.Fatal("opened object is not a directory")
	} else {
		directory = d
	}

	// Access the internal directory and ensure that doing so succeeds.
	if d, err := directory.OpenDirectory(longDirectoryName, logger); err != nil {
		t.Error("unable to open directory with long name:", err)
	} else {
		must.Close(d, logger)
	}

	// Access the internal file and ensure that doing so succeeds.
	if f, _, err := directory.OpenFile(longFileName); err != nil {
		t.Error("unable to open file with long name:", err)
	} else {
		must.Close(f, logger)
	}

	// Try to set permissions.
	self, err := user.Current()
	if err != nil {
		t.Fatal("unable to access current user:", err)
	}
	ownership, err := NewOwnershipSpecification("sid:"+self.Uid, "")
	if err != nil {
		t.Fatal("unable to construct ownership specification")
	}
	if err := directory.SetPermissions(longFileName, ownership, 0660); err != nil {
		t.Error("unable to set permissions for file with long name:", err)
	}

	// Ensure that removing the internal directory succeeds.
	if err := directory.RemoveDirectory(longDirectoryName); err != nil {
		t.Error("unable to remove directory with long name:", err)
	}

	// Ensure that removing the internal file succeeds.
	if err := directory.RemoveFile(longFileName); err != nil {
		t.Error("unable to remove file with long name:", err)
	}
}
