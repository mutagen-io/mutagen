package agent

import (
	"io"
	"os/exec"

	"github.com/pkg/errors"
)

// agentCloser implements io.Closer for agent processes.
type agentCloser struct {
	process *exec.Cmd
}

// Close "closes" the agent and unblocks its input/output streams.
// HACK: Rather than closing the process' standard input/output, this method
// simply terminates the agent process. The problem with closing the
// input/output streams is that they'll be OS pipes that might be blocked in
// reads or writes and won't necessarily unblock if closed, and they might even
// block the close - it's all platform dependent. But terminating the process
// will close the remote ends of the pipes and thus unblocks and reads/writes.
func (c *agentCloser) Close() error {
	// HACK: Accessing the Process field of an os/exec.Cmd could be a bit
	// dangerous if other code was accessing the Cmd at the same time, but in
	// our case the Cmd becomes completely encapsulated inside agentCloser
	// before agentCloser is returned, so it's okay.
	if c.process.Process != nil {
		if err := c.process.Process.Kill(); err != nil {
			return errors.Wrap(err, "unable to kill underlying process")
		}
	}
	return nil
}

func extractAgentStreams(process *exec.Cmd) (io.Writer, io.Reader, io.Reader, io.Closer, error) {
	// Redirect the process' standard input.
	standardInput, err := process.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "unable to redirect input")
	}

	// Redirect the process' standard output.
	standardOutput, err := process.StdoutPipe()
	if err != nil {
		standardInput.Close()
		return nil, nil, nil, nil, errors.Wrap(err, "unable to redirect output")
	}

	// Redirect the process' standard error output.
	standardError, err := process.StderrPipe()
	if err != nil {
		standardInput.Close()
		standardOutput.Close()
		return nil, nil, nil, nil, errors.Wrap(err, "unable to redirect error output")
	}

	// Create the result.
	return standardInput, standardOutput, standardError, &agentCloser{process}, nil
}
