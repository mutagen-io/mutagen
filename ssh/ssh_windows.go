package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/process"
)

// commandSearchPaths specifies additional locations on Windows where we might
// find ssh.exe and scp.exe binaries.
var commandSearchPaths = []string{
	`C:\Program Files\Git\usr\bin`,
	`C:\Program Files (x86)\Git\usr\bin`,
	`C:\msys32\usr\bin`,
	`C:\msys64\usr\bin`,
	// TODO: Add Cygwin binary paths.
}

func commandNamed(name string) (string, error) {
	// First, try to use the standard LookPath mechanism in case the user has
	// the binaries in their path already.
	if result, err := exec.LookPath(name); err == nil {
		return result, nil
	}

	// If there was nothing in the path, then scan other well-known directories
	// where we might find a viable binary.
	for _, path := range commandSearchPaths {
		target := filepath.Join(path, fmt.Sprintf("%s.exe", name))
		// TODO: Should we inspect the information used by stat to ensure this
		// is a file? No real need to check executability, anything with an exe
		// extension on Windows shows up as executable.
		if _, err := os.Stat(target); err != nil {
			return target, nil
		}
	}

	// Failure.
	return "", errors.New("unable to locate command")
}

func scpCommand() (string, error) {
	return commandNamed("scp")
}

func sshCommand() (string, error) {
	return commandNamed("ssh")
}

func processAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: process.DETACHED_PROCESS,
	}
}
