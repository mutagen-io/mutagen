package main

import (
	"io"
	"net"
	"os"
	"time"

	"github.com/pkg/errors"
)

// stdioAddress is a net.Addr implementation for stdioConnection.
type stdioAddress struct{}

// Network implements net.Addr.Network.
func (stdioAddress) Network() string {
	return "standard input/output"
}

// String implements net.Addr.String.
func (stdioAddress) String() string {
	return "standard input/output"
}

// stdioConnection is a net.Conn implementation that uses standard input/output.
type stdioConnection struct {
	io.Reader
	io.Writer
}

// newStdioConnection creates a new connection that uses standard input/output.
func newStdioConnection() net.Conn {
	return &stdioConnection{os.Stdin, os.Stdout}
}

// Close implements net.Conn.Close.
func (c *stdioConnection) Close() error {
	// We can't really close standard input/output, because on many platforms
	// these can't be unblocked on reads and writes, and they'll actually block
	// the call to close. In the case of the agent, where we're just running an
	// endpoint (which will have terminated by the time this connection has
	// closed), this is fine.
	//
	// Since we're not going to close the input/output streams, it doesn't make
	// sense to close the compressor and decompressor. Since these don't leak
	// any resources, this should be fine.
	return errors.New("closing standard input/output connection not allowed")
}

// LocalAddr implements net.Conn.LocalAddr.
func (c *stdioConnection) LocalAddr() net.Addr {
	return stdioAddress{}
}

// RemoteAddr implements net.Conn.RemoteAddr.
func (c *stdioConnection) RemoteAddr() net.Addr {
	return stdioAddress{}
}

// SetDeadline implements net.Conn.SetDeadline.
func (c *stdioConnection) SetDeadline(_ time.Time) error {
	return errors.New("deadlines not supported by standard input/output connections")
}

// SetReadDeadline implements net.Conn.SetReadDeadline.
func (c *stdioConnection) SetReadDeadline(_ time.Time) error {
	return errors.New("read deadlines not supported by standard input/output connections")
}

// SetWriteDeadline implements net.Conn.SetWriteDeadline.
func (c *stdioConnection) SetWriteDeadline(_ time.Time) error {
	return errors.New("write deadlines not supported by standard input/output connections")
}
