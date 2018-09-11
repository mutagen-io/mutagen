package process

import (
	"io"
	"net"
	"os/exec"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// address implements net.Addr for connection.
type address struct{}

// Network returns the connection protocol name.
func (_ address) Network() string {
	return "standard input/output"
}

// String returns the connection address.
func (_ address) String() string {
	return "standard input/output"
}

// connection implements net.Conn around the standard input/output of a process.
type connection struct {
	// process is the underlying process.
	process *exec.Cmd
	// standardOutput is the source for process output data.
	standardOutput io.Reader
	// standardInput is the destination for process input data.
	standardInput io.Writer
	// closeOnce is a one-time executor used to ensure that the underlying
	// process is only closed once.
	closeOnce sync.Once
}

// NewConnection creates a new net.Conn object by wraping a command object. It
// must be called before the corresponding process is started.
func NewConnection(process *exec.Cmd) (net.Conn, error) {
	// Redirect the process' standard input.
	standardInput, err := process.StdinPipe()
	if err != nil {
		return nil, errors.Wrap(err, "unable to redirect process input")
	}

	// Redirect the process' standard output.
	standardOutput, err := process.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "unable to redirect process output")
	}

	// Create the result.
	return &connection{
		process:        process,
		standardOutput: standardOutput,
		standardInput:  standardInput,
	}, nil
}

// Read reads from the process connection.
func (c *connection) Read(buffer []byte) (int, error) {
	return c.standardOutput.Read(buffer)
}

// Write writes to the process connection.
func (c *connection) Write(buffer []byte) (int, error) {
	return c.standardInput.Write(buffer)
}

// Close closes the process connection by terminating the underlying process.
// HACK: Rather than closing the process' standard input/output, this method
// simply terminates the process. The problem with closing the input/output
// streams is that they'll be OS pipes that might be blocked in reads or writes
// and won't necessarily unblock if closed, and they might even block the close
// - it's all platform dependent. But terminating the process will close the
// remote ends of the pipes and thus unblocks and reads/writes.
func (c *connection) Close() error {
	// Track errors.
	var err error

	// Terminate the underlying process, but only once.
	c.closeOnce.Do(func() {
		// HACK: Accessing the Process field of an os/exec.Cmd could be a bit
		// dangerous if other code was accessing the Cmd at the same time, but
		// in our use cases the Cmd becomes completely encapsulated inside the
		// connection before connection is returned, so it's okay.
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
func (c *connection) LocalAddr() net.Addr {
	return address{}
}

// RemoteAddr returns the remote address for the connection.
func (c *connection) RemoteAddr() net.Addr {
	return address{}
}

// SetDeadline sets the read and write deadlines for the connection.
func (c *connection) SetDeadline(_ time.Time) error {
	return errors.New("deadlines not supported by process connections")
}

// SetReadDeadline sets the read deadline for the connection.
func (c *connection) SetReadDeadline(_ time.Time) error {
	return errors.New("read deadlines not supported by process connections")
}

// SetWriteDeadline sets the write deadline for the connection.
func (c *connection) SetWriteDeadline(_ time.Time) error {
	return errors.New("write deadlines not supported by process connections")
}
