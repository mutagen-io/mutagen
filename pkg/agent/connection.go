package agent

import (
	"io"
	"net"
	"os/exec"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/compression"
)

// agentAddress implements net.Addr for agentConnection.
type agentAddress struct{}

// Network returns the connection protocol name.
func (_ agentAddress) Network() string {
	// TODO: Should we try to use URLs to give better information here?
	return "agent"
}

// String returns the connection address.
func (_ agentAddress) String() string {
	// TODO: Should we try to use URLs to give better information here?
	return "agent"
}

// agentConnection implements net.Conn around the standard input/output of an
// agent process.
type agentConnection struct {
	// process is the process hosting the connection to the agent.
	process *exec.Cmd
	// reader is the source for process output data. It may be either the raw
	// standard output or a flate decompressor which wraps this output.
	reader io.Reader
	// writer is the destination for process input data. It may be either the
	// raw standard input or an automatically flushing flate compressor which
	// wraps this input.
	writer io.Writer
	// closeOnce is a one-time executor used to ensure that the underlying
	// process is only closed once.
	closeOnce sync.Once
}

// newAgentConnection creates a new net.Conn object by wraping an agent process.
// It must be called before the process is started. It optionally supports
// compression.
func newAgentConnection(process *exec.Cmd, compress bool) (net.Conn, error) {
	// Redirect the process' standard input and optionally wrap it in a
	// compressor.
	var writer io.Writer
	standardInput, err := process.StdinPipe()
	if err != nil {
		return nil, errors.Wrap(err, "unable to redirect process input")
	}
	if compress {
		writer = compression.NewCompressingWriter(standardInput)
	} else {
		writer = standardInput
	}

	// Redirect the process' standard output and optionally wrap it in a
	// decompressor.
	var reader io.Reader
	standardOutput, err := process.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "unable to redirect process output")
	}
	if compress {
		reader = compression.NewDecompressingReader(standardOutput)
	} else {
		reader = standardOutput
	}

	// Create the result.
	return &agentConnection{
		process: process,
		reader:  reader,
		writer:  writer,
	}, nil
}

// Read reads from the agent connection.
func (c *agentConnection) Read(buffer []byte) (int, error) {
	return c.reader.Read(buffer)
}

// Write writes to the agent connection.
func (c *agentConnection) Write(buffer []byte) (int, error) {
	return c.writer.Write(buffer)
}

// Close closes the agent stream.
// HACK: Rather than closing the process' standard input/output, this method
// simply terminates the agent process. The problem with closing the
// input/output streams is that they'll be OS pipes that might be blocked in
// reads or writes and won't necessarily unblock if closed, and they might even
// block the close - it's all platform dependent. But terminating the process
// will close the remote ends of the pipes and thus unblocks and reads/writes.
// HACK: As a result of simply terminating the process, we also don't close the
// compressor and decompressor, however these don't leak any resources if left
// unclosed, so that shouldn't be a problem. By not doing this, we're relying on
// an implementation detail of the flate package, but since the interface of the
// flate package limits itself to accepting io.Reader/Writer, it limits what
// could possibly leak. This isn't ideal, but it mirrors the behavior on the
// agent side of things, and it's necessary to avoid any of the side-effects of
// those Close methods (like trying to read/write on the underlying stream,
// which can lead to indefinite blocking for OS pipes).
func (c *agentConnection) Close() error {
	// Track errors.
	var err error

	// Terminate the underlying process, but only once.
	c.closeOnce.Do(func() {
		// HACK: Accessing the Process field of an os/exec.Cmd could be a bit
		// dangerous if other code was accessing the Cmd at the same time, but in
		// our case the Cmd becomes completely encapsulated inside agentConnection
		// before agentConnection is returned, so it's okay.
		if c.process.Process != nil {
			err = c.process.Process.Kill()
		}
	})

	// Handle errors.
	if err != nil {
		return errors.Wrap(err, "unable to kill underlying process")
	}

	// Done.
	return nil
}

// LocalAddr returns the local address for the connection.
func (c *agentConnection) LocalAddr() net.Addr {
	return agentAddress{}
}

// RemoteAddr returns the remote address for the connection.
func (c *agentConnection) RemoteAddr() net.Addr {
	return agentAddress{}
}

// SetDeadline sets the read and write deadlines for the connection.
func (c *agentConnection) SetDeadline(_ time.Time) error {
	return errors.New("deadlines not supported by agent connections")
}

// SetReadDeadline sets the read deadline for the connection.
func (c *agentConnection) SetReadDeadline(_ time.Time) error {
	return errors.New("read deadlines not supported by agent connections")
}

// SetWriteDeadline sets the write deadline for the connection.
func (c *agentConnection) SetWriteDeadline(_ time.Time) error {
	return errors.New("write deadlines not supported by agent connections")
}
