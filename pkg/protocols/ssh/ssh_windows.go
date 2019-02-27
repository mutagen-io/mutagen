package ssh

import (
	"github.com/havoc-io/mutagen/pkg/process"
)

// commandSearchPaths specifies locations on Windows where we might find ssh.exe
// and scp.exe binaries.
var commandSearchPaths = []string{
	// TODO: Add the PowerShell OpenSSH paths at the top of this list once
	// there's a usable release.
	`C:\Program Files\Git\usr\bin`,
	`C:\Program Files (x86)\Git\usr\bin`,
	`C:\msys32\usr\bin`,
	`C:\msys64\usr\bin`,
	`C:\cygwin\bin`,
	`C:\cygwin64\bin`,
}

// sshCommandForPlatform will search for a suitable ssh command implementation
// on Windows.
func sshCommandForPlatform() (string, error) {
	return process.FindCommand("ssh", commandSearchPaths)
}

// scpCommandForPlatform will search for a suitable scp command implementation
// on Windows.
func scpCommandForPlatform() (string, error) {
	return process.FindCommand("scp", commandSearchPaths)
}
