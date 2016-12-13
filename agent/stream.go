package agent

import (
	"io"
	"os/exec"
)

// processStream implements io.ReadWriteCloser around the standard input/output
// of a process.
type processStream struct {
	process        *exec.Cmd
	standardInput  io.WriteCloser
	standardOutput io.Reader
}

func (s *processStream) Read(p []byte) (int, error) {
	return s.standardOutput.Read(p)
}

func (s *processStream) Write(p []byte) (int, error) {
	return s.standardInput.Write(p)
}

func (s *processStream) Close() error {
	// We don't wait for the process to exit because it will exit anyway once
	// standard input is closed (and won't exit if standard input can't be
	// closed, which would be rare but block indefinitely in a wait).
	return s.standardInput.Close()
}
