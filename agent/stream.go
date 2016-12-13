package agent

import (
	"io"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/stream"
)

// processStream implements io.ReadWriteCloser around the standard input/output
// of a process.
type processStream struct {
	process *exec.Cmd
	io.ReadWriteCloser
}

func newProcessStream(process *exec.Cmd) (io.ReadWriteCloser, error) {
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
	return &processStream{
		process,
		stream.NewStream(standardOutput, standardInput, standardInput),
	}, nil
}
