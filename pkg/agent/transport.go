package agent

import (
	"os/exec"

	"github.com/pkg/errors"
)

// Transport is the interface that protocols wishing to use the agent
// infrastructure must implement. It should be natural for "SSH"-like protocols,
// i.e. those where SCP-like and SSH-like operations can be performed. Transport
// instances do not need to be safe for concurrent invocation unless being used
// for concurrent Dial operations.
type Transport interface {
	// Copy copies the specified local file (which is guaranteed to exist and be
	// a file) to the remote. The provided local path will be absolute. The
	// remote path will be a filename (i.e. without path separators) that should
	// be treated as being relative to the user's home directory.
	Copy(localPath, remoteName string) error
	// Command creates (but does not start) a process that will invoke the
	// specified command on the specified remote. It should not re-direct any of
	// the output streams of the process. The command on the remote must be
	// invoked with the user's home directory as the working directory. Any
	// command provided to this interface is guaranteed to be lexable by simply
	// splitting on spaces.
	Command(command string) (*exec.Cmd, error)
}

// run is a utility method that invoke's a command via a transport, waits for it
// to complete, and returns its exit error. If there is an error creating the
// command, it will be returned wrapped, but otherwise the result of the run
// method will be returned un-wrapped, so it can be treated as an
// os/exec.ExitError.
func run(transport Transport, command string) error {
	// Create the process.
	process, err := transport.Command(command)
	if err != nil {
		return errors.Wrap(err, "unable to create command")
	}

	// Run the process.
	return process.Run()
}

// output is a utility method that invoke's a command via a transport, waits for
// it to complete, and returns its standard output and exit error. If there is
// an error creating the command, it will be returned wrapped, but otherwise the
// result of the run method will be returned un-wrapped, so it can be treated as
// an os/exec.ExitError.
func output(transport Transport, command string) ([]byte, error) {
	// Create the process.
	process, err := transport.Command(command)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create command")
	}

	// Run the process.
	return process.Output()
}
