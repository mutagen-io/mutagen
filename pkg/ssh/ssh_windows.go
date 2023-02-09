package ssh

import (
	"github.com/mutagen-io/mutagen/pkg/platform"
)

// commandSearchPaths specifies locations on Windows where we might find ssh.exe
// and scp.exe binaries.
var commandSearchPaths = []string{
	`C:\Program Files\Git\usr\bin`,
	`C:\Program Files (x86)\Git\usr\bin`,
	`C:\msys32\usr\bin`,
	`C:\msys64\usr\bin`,
	`C:\cygwin\bin`,
	`C:\cygwin64\bin`,
	`C:\Windows\System32\OpenSSH`,
}

// sshCommandPathForPlatform will search for a suitable ssh command on Windows.
func sshCommandPathForPlatform() (string, error) {
	return platform.FindCommand("ssh", commandSearchPaths)
}

// scpCommandPathForPlatform will search for a suitable scp command on Windows.
func scpCommandPathForPlatform() (string, error) {
	return platform.FindCommand("scp", commandSearchPaths)
}
