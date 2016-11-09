package agent

import (
	"io"
	"net"
	"time"

	"github.com/pkg/errors"

	"google.golang.org/grpc"
)

type addr struct {
	name string
}

func (a *addr) Network() string {
	return a.name
}

func (a *addr) String() string {
	return a.name
}

type ioConn struct {
	input             io.Reader
	output            io.Writer
	closers           []io.Closer
	terminationMarked bool
	termination       chan<- struct{}
}

func NewIOConn(input io.Reader, output io.Writer, closers ...io.Closer) (net.Conn, <-chan struct{}) {
	// Create the termination channel.
	termination := make(chan struct{})

	// Create the connection.
	return &ioConn{
		input:       input,
		output:      output,
		closers:     closers,
		termination: termination,
	}, termination
}

func (c *ioConn) markTermination() {
	// If we've already marked termination by closing the channel, don't do it
	// again.
	if c.terminationMarked {
		return
	}

	// Mark termination recorded and signal by closing the channel.
	c.terminationMarked = true
	close(c.termination)
}

func (c *ioConn) Read(b []byte) (int, error) {
	// Forward the read.
	n, err := c.input.Read(b)

	// If any error occurred, mark termination.
	if err != nil {
		c.markTermination()
	}

	// Done.
	return n, err
}

func (c *ioConn) Write(b []byte) (int, error) {
	// Forward the write to standard output.
	n, err := c.output.Write(b)

	// If any error occurred, mark termination.
	if err != nil {
		c.markTermination()
	}

	// Done.
	return n, err
}

// Close attempts to implement the net.Conn.Close method, though it may not
// share the exact same semantics because the underlying streams may not support
// unblocking Read/Write calls on Close. In this case, you have to be very
// careful to avoid relying on this preemption behavior, otherwise the calling
// Goroutines might deadlock.
func (c *ioConn) Close() error {
	// Close all closers, marking the first error.
	var err error
	for _, closer := range c.closers {
		if closeErr := closer.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}

	// Done.
	return err
}

func (c *ioConn) LocalAddr() net.Addr {
	return &addr{"io"}
}

func (c *ioConn) RemoteAddr() net.Addr {
	return &addr{"io"}
}

func (c *ioConn) SetDeadline(_ time.Time) error {
	return errors.New("deadlines not supported")
}

func (c *ioConn) SetReadDeadline(_ time.Time) error {
	return errors.New("deadlines not supported")
}

func (c *ioConn) SetWriteDeadline(_ time.Time) error {
	return errors.New("deadlines not supported")
}

type oneShotListener struct {
	conn net.Conn
}

func NewOneShotListener(conn net.Conn) net.Listener {
	return &oneShotListener{conn}
}

func (l *oneShotListener) Accept() (net.Conn, error) {
	// If a connection is present, nil out the record of it and return it.
	if l.conn != nil {
		conn := l.conn
		l.conn = nil
		return conn, nil
	}

	// If there are no connections, we're done.
	return nil, errors.New("no more connections")
}

func (l *oneShotListener) Close() error {
	// No accept calls ever block, so we don't need to do anything.
	return nil
}

func (l *oneShotListener) Addr() net.Addr {
	return &addr{"memory"}
}

type oneShotDialerError struct{}

func (e *oneShotDialerError) Error() string {
	return "dialer is one-shot"
}

func (e *oneShotDialerError) Temporary() bool {
	return false
}

func clientWithConn(conn net.Conn) *grpc.ClientConn {
	// Create a one-shot dialer to use in client creation. This dialer will
	// return an error if invoked more than once, and gRPC will recognize that
	// error as non-temporary, thereby aborting any redials.
	// TODO: This behavior relies on the following PR being merged:
	// https://github.com/grpc/grpc-go/pull/974
	conns := make(chan net.Conn, 1)
	conns <- conn
	close(conns)
	dialer := func(_ string, _ time.Duration) (net.Conn, error) {
		if c, ok := <-conns; ok {
			return c, nil
		}
		return nil, &oneShotDialerError{}
	}

	// Perform a dial, enforcing that this work the first time through.
	client, err := grpc.Dial("", grpc.WithBlock(), grpc.WithDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic(errors.Wrap(err, "in-memory dial failed"))
	}

	// Success.
	return client
}
