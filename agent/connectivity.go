package agent

import (
	"io"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/pkg/errors"
)

type stdioAddr struct{}

func (_ *stdioAddr) Network() string {
	return "stdio"
}

func (_ *stdioAddr) String() string {
	return "stdio"
}

type stdioConn struct{}

func (_ *stdioConn) Read(b []byte) (int, error) {
	return os.Stdin.Read(b)
}

func (_ *stdioConn) Write(b []byte) (int, error) {
	return os.Stdout.Write(b)
}

// Close does NOT implement the net.Conn.Close method. This is unfortunately not
// possible with standard input/output because calling Close on those files
// might block if they are being read to or written from. This can very easily
// lead to a deadlock if no more input is coming or no more output is going to
// be processed. Unfortunately there is no way to implement net.Conn.Close
// semantics (which are supposed to unblock Read/Write operations) with standard
// input/output. For this connection, which is effectively a singleton and will
// only be used once and for the lifetime of the process, it's best to just
// "close" it by simply exiting the process.
func (_ *stdioConn) Close() error {
	panic("standard input/output connections don't support closing")
}

func (_ *stdioConn) LocalAddr() net.Addr {
	return &stdioAddr{}
}

func (_ *stdioConn) RemoteAddr() net.Addr {
	return &stdioAddr{}
}

func (_ *stdioConn) SetDeadline(_ time.Time) error {
	return errors.New("deadlines not supported")
}

func (_ *stdioConn) SetReadDeadline(_ time.Time) error {
	return errors.New("deadlines not supported")
}

func (_ *stdioConn) SetWriteDeadline(_ time.Time) error {
	return errors.New("deadlines not supported")
}

type stdioListener struct {
	conns chan net.Conn
}

func NewStdioListener() net.Listener {
	// Create a connections channel, with enough space for our lone connection.
	conns := make(chan net.Conn, 1)

	// Populate the connections.
	conns <- &stdioConn{}

	// Create the listener.
	return &stdioListener{
		conns: conns,
	}
}

func (l *stdioListener) Accept() (net.Conn, error) {
	// Grab the next connection.
	conn, ok := <-l.conns

	// If it was already consumed, we've probably been triggered due to close.
	if !ok {
		return nil, errors.New("listener closed")
	}

	// Success.
	return conn, nil
}

func (l *stdioListener) Close() error {
	// Close the connections channel, terminating any Accept calls.
	close(l.conns)

	// Success.
	return nil
}

func (l *stdioListener) Addr() net.Addr {
	return &stdioAddr{}
}

type agentConn struct {
	process *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.Reader
}

func (c *agentConn) Read(p []byte) (int, error) {
	return c.stdout.Read(p)
}

func (c *agentConn) Write(p []byte) (int, error) {
	return c.stdin.Write(p)
}

func (c *agentConn) Close() error {
	// Close the process' standard input.
	if err := c.stdin.Close(); err != nil {
		c.process.Wait()
		return err
	}

	// Wait for the process to terminate.
	return c.process.Wait()
}

func (c *agentConn) LocalAddr() net.Addr {
	return &stdioAddr{}
}

func (c *agentConn) RemoteAddr() net.Addr {
	return &stdioAddr{}
}

func (c *agentConn) SetDeadline(_ time.Time) error {
	return errors.New("deadlines not supported")
}

func (c *agentConn) SetReadDeadline(_ time.Time) error {
	return errors.New("deadlines not supported")
}

func (c *agentConn) SetWriteDeadline(_ time.Time) error {
	return errors.New("deadlines not supported")
}
