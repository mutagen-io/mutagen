package agent

import (
	"io"
	"os/exec"

	"github.com/pkg/errors"
)

// agentStream implements io.ReadWriteCloser around the standard input/output of
// an agent process.
type agentStream struct {
	process *exec.Cmd
	io.Reader
	io.Writer
}

// Close closes the agent stream.
// HACK: Rather than closing the process' standard input/output, this method
// simply terminates the agent process. The problem with closing the
// input/output streams is that they'll be OS pipes that might be blocked in
// reads or writes and won't necessarily unblock if closed, and they might even
// block the close - it's all platform dependent. But terminating the process
// will close the remote ends of the pipes and thus unblocks and reads/writes.
func (s *agentStream) Close() error {
	// HACK: Accessing the Process field of an os/exec.Cmd could be a bit
	// dangerous if other code was accessing the Cmd at the same time, but in
	// our case the Cmd becomes completely encapsulated inside agentStream
	// before agentStream is returned, so it's okay.
	if s.process.Process != nil {
		if err := s.process.Process.Kill(); err != nil {
			return errors.Wrap(err, "unable to kill underlying process")
		}
	}
	return nil
}

func newAgentStream(process *exec.Cmd) (io.ReadWriteCloser, error) {
	// Redirect the process' standard input.
	standardInput, err := process.StdinPipe()
	if err != nil {
		return nil, errors.Wrap(err, "unable to redirect input")
	}

	// Redirect the process' standard output.
	standardOutput, err := process.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "unable to redirect output")
	}

	// Create the result.
	return &agentStream{process, standardOutput, standardInput}, nil
}
