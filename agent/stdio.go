package agent

import (
	"net"
	"os"
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

func (_ *stdioConn) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
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
