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
	// killDelay specifies the duration that the connection should wait for the
	// underlying process to exit on its own before killing the process.
	killDelay time.Duration
}

// NewConnection creates a new net.Conn object by wraping a command object. It
// must be called before the corresponding process is started.
func NewConnection(process *exec.Cmd, killDelay time.Duration) (net.Conn, error) {
	// Validate the kill delay time.
	if killDelay < time.Duration(0) {
		panic("negative kill delay specified")
	}

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
		killDelay:      killDelay,
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

// Close closes the process connection by terminating the underlying process and
// waiting for it to exit. If a non-negative/non-zero kill delay has been
// specified, then this method will wait (up to the specified duration) for the
// process to exit on its own before issuing a kill request. By the time this
// method returns, the underlying process is guaranteed to no longer be running.
// HACK: Rather than closing the process' standard input/output, this method
// simply terminates the process. The problem with closing the input/output
// streams is that they'll be OS pipes that might be blocked in reads or writes
// and won't necessarily unblock if closed, and they might even block the close
// - it's all platform dependent. But terminating the process will close the
// remote ends of the pipes and thus unblocks and reads/writes.
func (c *connection) Close() error {
	// Verify that the process was actually started.
	if c.process.Process == nil {
		return errors.New("process not started")
	}

	// Start a background Goroutine that will wait for the process to exit and
	// return the wait result.
	waitResults := make(chan error, 1)
	go func() {
		waitResults <- c.process.Wait()
	}()

	// Wait, up to the specified duration, for the process to exit on its own.
	select {
	case err := <-waitResults:
		return errors.Wrap(err, "process wait failed")
	case <-time.After(c.killDelay):
	}

	// Issue a kill request.
	// HACK: We don't handle errors here, because there's not much we can do
	// with the information. We need to guarantee that, by the time this method
	// returns, the process is no longer running. That will be enforced by our
	// indefinite wait in the return statement, but it's possible that the kill
	// signal could fail, and that the process could run indefinitely. That's
	// highly unlikely though, and it's safer to block indefinitely in that case
	// than to return with an error that might not be checked.
	c.process.Process.Kill()

	// Wait for the wait operation to complete.
	return errors.Wrap(<-waitResults, "process wait failed")
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
