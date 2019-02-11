package ssh

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/url"
)

func TestSCPCommand(t *testing.T) {
	if commandName, err := scpCommand(); err != nil {
		t.Fatal("unable to locate SCP command:", err)
	} else if commandName == "" {
		t.Error("SCP command name is empty")
	}
}

func TestSSHCommand(t *testing.T) {
	if commandName, err := sshCommand(); err != nil {
		t.Fatal("unable to locate SSH command:", err)
	} else if commandName == "" {
		t.Error("SSH command name is empty")
	}
}

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

	// Compute the remote URL.
	remote := &url.URL{
		Protocol: url.Protocol_SSH,
		Username: user.Username,
		Hostname: "localhost",
		Port:     22,
		Path:     "~/synchronization/path",
	}

	// Create a transport.
	transport := &transport{remote: remote}

	// Compute the destination path.
	// HACK: Technically agent.Transport implementations only need to support
	// remote destination paths that are file names relative to the home
	// directory. For testing, however, we don't want to copy into the home
	// directory, and since we know our Copy implementation can support
	// arbitrary remote paths, we use one.
	destination := filepath.Join(directory, "destination")

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

	// Compute remote URL.
	remote := &url.URL{
		Protocol: url.Protocol_SSH,
		Username: user.Username,
		Hostname: "localhost",
		Port:     22,
		Path:     "~/synchronization/path",
	}

	// Create a transport.
	transport := &transport{remote: remote}

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
