package agent

import (
	"io"
	"net"
	"os/exec"
	"time"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/url"
)

// agentAddress implements net.Addr for agentConnection.
type agentAddress struct {
	// local encodes whether or not the address is behaving as a local or remote
	// address.
	local bool
	// url is the remote URL for the agent.
	url *url.URL
}

// Network returns the name of the agent protocol being used.
func (a *agentAddress) Network() string {
	// See if this is a protocol known to the agent package.
	if a.url.Protocol == url.Protocol_SSH {
		return "ssh"
	}

	// If not, just return a default value.
	return "unknown"
}

// String returns the URL for the agent for remote addresses and a "local" for
// local addresses.
func (a *agentAddress) String() string {
	// If this is a local address, the remote URL doesn't apply.
	if a.local {
		return "local"
	}

	// If this is a remote address, return the URL.
	return a.url.Format()
}

// agentConnection implements net.Conn around the standard input/output of an
// agent process.
type agentConnection struct {
	// url is the remote URL used to connect to the agent.
	url *url.URL
	// process is the process hosting the connection to the agent.
	process *exec.Cmd
	// Reader is the process' standard output.
	io.Reader
	// Writer is the process' standard input.
	io.Writer
}

// Close closes the agent stream.
// HACK: Rather than closing the process' standard input/output, this method
// simply terminates the agent process. The problem with closing the
// input/output streams is that they'll be OS pipes that might be blocked in
// reads or writes and won't necessarily unblock if closed, and they might even
// block the close - it's all platform dependent. But terminating the process
// will close the remote ends of the pipes and thus unblocks and reads/writes.
func (c *agentConnection) Close() error {
	// HACK: Accessing the Process field of an os/exec.Cmd could be a bit
	// dangerous if other code was accessing the Cmd at the same time, but in
	// our case the Cmd becomes completely encapsulated inside agentConnection
	// before agentConnection is returned, so it's okay.
	if c.process.Process != nil {
		if err := c.process.Process.Kill(); err != nil {
			return errors.Wrap(err, "unable to kill underlying process")
		}
	}

	// Done.
	return nil
}

// LocalAddr returns the local address for the connection.
func (c *agentConnection) LocalAddr() net.Addr {
	return &agentAddress{true, c.url}
}

// RemoteAddr returns the remote address for the connection.
func (c *agentConnection) RemoteAddr() net.Addr {
	return &agentAddress{false, c.url}
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

// newAgentConnection creates a new net.Conn object by wraping an agent process.
// It must be called before the process is stated.
func newAgentConnection(url *url.URL, process *exec.Cmd) (net.Conn, error) {
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
	return &agentConnection{url, process, standardOutput, standardInput}, nil
}
