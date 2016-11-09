package agent

import (
	"net"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen"
	processpkg "github.com/havoc-io/mutagen/process"
	"github.com/havoc-io/mutagen/url"
)

const (
	errorCodeCommandNotFound = 127
)

type sshConn struct {
	net.Conn
	process *exec.Cmd
}

func connectSSH(prompter string, remote *url.SSHURL) (net.Conn, bool, error) {
	// Create an SSH process.
	process, err := ssh(prompter, "Connecting to agent", remote, agentSSHCommand)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to create SSH command")
	}

	// Create pipes to the process.
	stdin, err := process.StdinPipe()
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to redirect SSH input")
	}
	stdout, err := process.StdoutPipe()
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to redirect SSH output")
	}

	// Start the process.
	if err = process.Start(); err != nil {
		return nil, false, errors.Wrap(err, "unable to start SSH process")
	}

	// Confirm that the process started correctly by performing a version
	// handshake.
	// TODO: Figure out how to identify "command not found" errors for Windows
	// SSH servers.
	if versionMatch, err := mutagen.ReceiveAndCompareVersion(stdout); err != nil {
		code, codeErr := processpkg.ExitCodeForError(process.Wait())
		if codeErr == nil && code == errorCodeCommandNotFound {
			return nil, true, errors.New("command not found")
		} else {
			return nil, false, errors.Wrap(err, "unable to handshake with SSH process")
		}
	} else if !versionMatch {
		return nil, true, errors.New("version mismatch")
	}

	// Create a connection.
	// HACK: TODO: Document why we don't register stdout as a closer.
	conn, _ := NewIOConn(stdout, stdin, stdin)
	return &sshConn{conn, process}, false, nil
}
