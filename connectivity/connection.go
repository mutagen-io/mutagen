package connectivity

import (
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
)

type ioConnection struct {
	input             io.Reader
	output            io.Writer
	closers           []io.Closer
	terminationMarked bool
	termination       chan<- struct{}
}

// NewIOConnection returns a net.Conn that wraps the specified Reader, Writer,
// and Closers. See an important note on the Close method regarding blocking
// behavior. This method also returns a channel that will be closed when either
// Read or Write returns any error.
func NewIOConnection(input io.Reader, output io.Writer, closers ...io.Closer) (net.Conn, <-chan struct{}) {
	// Create the termination channel.
	termination := make(chan struct{})

	// Create the connection.
	return &ioConnection{
		input:       input,
		output:      output,
		closers:     closers,
		termination: termination,
	}, termination
}

func (c *ioConnection) markTermination() {
	// If we've already marked termination by closing the channel, don't do it
	// again.
	if c.terminationMarked {
		return
	}

	// Mark termination recorded and signal by closing the channel.
	c.terminationMarked = true
	close(c.termination)
}

func (c *ioConnection) Read(b []byte) (int, error) {
	// Forward the read.
	n, err := c.input.Read(b)

	// If any error occurred, mark termination.
	if err != nil {
		c.markTermination()
	}

	// Done.
	return n, err
}

func (c *ioConnection) Write(b []byte) (int, error) {
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
func (c *ioConnection) Close() error {
	// Close all closers, recording the first error, if any.
	var err error
	for _, closer := range c.closers {
		if closeErr := closer.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}

	// Done.
	return err
}

func (c *ioConnection) LocalAddr() net.Addr {
	return newNamedAddress("io")
}

func (c *ioConnection) RemoteAddr() net.Addr {
	return newNamedAddress("io")
}

func (c *ioConnection) SetDeadline(_ time.Time) error {
	return errors.New("deadlines not supported")
}

func (c *ioConnection) SetReadDeadline(_ time.Time) error {
	return errors.New("deadlines not supported")
}

func (c *ioConnection) SetWriteDeadline(_ time.Time) error {
	return errors.New("deadlines not supported")
}
