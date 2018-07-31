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
	if command, err := scpCommand(); err != nil {
		t.Fatal("unable to locate SCP command:", err)
	} else if command == "" {
		t.Error("SCP command is empty")
	}
}

func TestSSHCommand(t *testing.T) {
	if command, err := sshCommand(); err != nil {
		t.Fatal("unable to locate SSH command:", err)
	} else if command == "" {
		t.Error("SSH command is empty")
	}
}

func TestProcessAttributes(t *testing.T) {
	if processAttributes() == nil {
		t.Error("nil process attributes returned")
	}
}

func TestCopyNonSSHURL(t *testing.T) {
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

	// Compute target URL.
	target := &url.URL{
		Protocol: url.Protocol_Local,
		Path:     filepath.Join(directory, "target"),
	}

	// Ensure the copy fails.
	if Copy("", source, target) == nil {
		t.Fatal("copy succeeded for non-SSH URL")
	}
}

func TestCopyRelativePath(t *testing.T) {
	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_ssh_copy")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Compute the current working directory.
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal("unable to compute working directory:", err)
	}

	// Compute source path and make it relative to the current directory.
	source := filepath.Join(directory, "source")
	if s, err := filepath.Rel(workingDirectory, source); err != nil {
		t.Fatal("unable to compute relative source path:", err)
	} else {
		source = s
	}

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

	// Ensure the copy fails.
	if Copy("", source, target) == nil {
		t.Fatal("copy succeeded for relative source path")
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

	// Compute target URL.
	target := &url.URL{
		Protocol: url.Protocol_SSH,
		Username: user.Username,
		Hostname: "localhost",
		Port:     22,
		Path:     filepath.Join(directory, "target"),
	}

	// Copy the file.
	if err := Copy("", source, target); err != nil {
		t.Fatal("unable to copy file:", err)
	}

	// Verify that the file exists.
	if _, err := os.Lstat(target.Path); err != nil {
		t.Error("unable to verify that target exists")
	}
}

func TestRunNonSSHURL(t *testing.T) {
	// Compute a command to run.
	command := "env"
	if runtime.GOOS == "windows" {
		command = "cmd /c set"
	}

	// Attempt to run.
	if Run("", &url.URL{}, command) == nil {
		t.Fatal("run succeeded for non-SSH URL")
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
	if err := Run("", remote, command); err != nil {
		t.Fatal("unable to run command")
	}
}

func TestOutputNonSSHURL(t *testing.T) {
	// Compute a command to run.
	command := "env"
	if runtime.GOOS == "windows" {
		command = "cmd /c set"
	}

	// Attempt to capture output.
	if _, err := Output("", &url.URL{}, command); err == nil {
		t.Fatal("output succeeded for non-SSH URL")
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
	if output, err := Output("", remote, command); err != nil {
		t.Fatal("unable to run command")
	} else if !strings.Contains(string(output), content) {
		t.Error("output does not contain expected content")
	} else if !utf8.Valid(output) {
		t.Error("output not in UTF-8 encoding")
	}
}
