package ssh

import (
	"fmt"
	"os"
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
	`C:\cygwin\bin`,
	`C:\cygwin64\bin`,
	// TODO: Add the PowerShell OpenSSH paths once there's a stable release.
}

func commandNamed(name string) (string, error) {
	// TODO: When the OpenSSH landscape on Windows eventually stablizes (i.e.
	// once the PowerShell team releases a stable OpenSSH version), we might try
	// to do an exec.LookPath call to let any binary in the user's path be
	// picked up first. We'd still need the well-known paths though, since they
	// might not be in the user's path.

	// Scan well-known directories where we might find a viable binary.
	for _, path := range commandSearchPaths {
		target := filepath.Join(path, fmt.Sprintf("%s.exe", name))
		// TODO: Should we inspect the information used by stat to ensure this
		// is a file? No real need to check executability, anything with an exe
		// extension on Windows shows up as executable.
		if _, err := os.Stat(target); err == nil {
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

func controlMasterArguments() []string {
	return nil
}
