package ssh

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

func TestCopy(t *testing.T) {
	// If localhost SSH support isn't available, then skip this test.
	if os.Getenv("MUTAGEN_TEST_SSH") != "true" {
		t.Skip()
	}

	// Compute source path.
	source := filepath.Join(t.TempDir(), "source")

	// Create contents.
	contents := []byte{0, 1, 2, 3, 4, 5, 6}

	// Attempt to write to a temporary file.
	if err := filesystem.WriteFileAtomic(source, contents, 0600); err != nil {
		t.Fatal("atomic file write failed:", err)
	}

	// Grab our username.
	user, err := user.Current()
	if err != nil {
		t.Fatal("unable to query user data:", err)
	}

	// Create a transport.
	transport := &transport{
		user: user.Username,
		host: "localhost",
		port: 22,
	}

	// Compute the destination path.
	// HACK: Technically agent.Transport implementations only need to support
	// remote destination paths that are file names relative to the home
	// directory. For testing, however, we don't want to copy into the home
	// directory, and since we know our Copy implementation can support
	// arbitrary remote paths, we use one.
	destination := filepath.Join(t.TempDir(), "destination")

	// Copy the file.
	if err := transport.Copy(source, destination); err != nil {
		t.Fatal("unable to copy file:", err)
	}

	// Verify that the file exists.
	if _, err := os.Lstat(destination); err != nil {
		t.Error("unable to verify that destination exists")
	}
}

func TestCommandOutput(t *testing.T) {
	// If localhost SSH support isn't available, then skip this test.
	if os.Getenv("MUTAGEN_TEST_SSH") != "true" {
		t.Skip()
	}

	// Compute a command to run.
	command := "env"
	if runtime.GOOS == "windows" {
		command = "cmd /c set"
	}

	// Compute expected output content.
	content := "HOME="
	if runtime.GOOS == "windows" {
		content = "PROCESSOR_ARCHITECTURE="
	}

	// Grab our username.
	user, err := user.Current()
	if err != nil {
		t.Fatal("unable to query user data:", err)
	}

	// Create a transport.
	transport := &transport{
		user: user.Username,
		host: "localhost",
		port: 22,
	}

	// Attempt to execute the command.
	// TODO: Should we also verify that an extracted HOME/USERPROFILE value
	// matches the expected home directory since we've already queried the user?
	if command, err := transport.Command(command); err != nil {
		t.Fatal("unable to create command:", err)
	} else if output, err := command.Output(); err != nil {
		t.Fatal("unable to run command:", err)
	} else if !strings.Contains(string(output), content) {
		t.Error("output does not contain expected content")
	} else if !utf8.Valid(output) {
		t.Error("output not in UTF-8 encoding")
	}
}
