package ssh

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/url"
)

func TestCopy(t *testing.T) {
	// If localhost SSH support isn't available, then skip this test.
	if os.Getenv("MUTAGEN_TEST_SSH") != "true" {
		t.Skip()
	}

	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_ssh_copy")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Compute source path.
	source := filepath.Join(directory, "source")

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

	// Compute target URL.
	target := &url.URL{
		Protocol: url.Protocol_SSH,
		Username: user.Username,
		Hostname: "localhost",
		Port:     22,
		Path:     filepath.Join(directory, "target"),
	}

	// Copy the file.
	if err := Copy("", "Copying file", source, target); err != nil {
		t.Fatal("unable to copy file:", err)
	}

	// Verify that the file exists.
	if _, err := os.Lstat(target.Path); err != nil {
		t.Error("unable to verify that target exists")
	}
}

func TestRun(t *testing.T) {
	// If localhost SSH support isn't available, then skip this test.
	if os.Getenv("MUTAGEN_TEST_SSH") != "true" {
		t.Skip()
	}

	// Compute a command to run.
	command := "env"
	if runtime.GOOS == "windows" {
		command = "cmd /c set"
	}

	// Grab our username.
	user, err := user.Current()
	if err != nil {
		t.Fatal("unable to query user data:", err)
	}

	// Compute remote URL.
	remote := &url.URL{
		Protocol: url.Protocol_SSH,
		Username: user.Username,
		Hostname: "localhost",
		Port:     22,
	}

	// Attempt to execute the command.
	if err := Run("", "Running command", remote, command); err != nil {
		t.Fatal("unable to run command")
	}
}

func TestOutput(t *testing.T) {
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

	// Compute remote URL.
	remote := &url.URL{
		Protocol: url.Protocol_SSH,
		Username: user.Username,
		Hostname: "localhost",
		Port:     22,
	}

	// Attempt to execute the command.
	if output, err := Output("", "Grabbing output", remote, command); err != nil {
		t.Fatal("unable to run command")
	} else if !strings.Contains(string(output), content) {
		t.Error("output does not contain expected content")
	}
}
