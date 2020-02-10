package monitor

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// monitoredConnection is a wrapper around net.Conn that monitors for and
// reports the first read or write error to occur.
type monitoredConnection struct {
	// connection is the underlying connection.
	connection net.Conn
	// report gates access to the error channel.
	report sync.Once
	// errors is the error channel.
	errors chan error
}

// Enable wraps the specified connection and creates an error channel that
// returns the first read or write error to occur on the channel (if any). This
// includes any deadline-induced errors. If no error ever occurs during a read
// or write, no error will be written to the channel.
func Enable(connection net.Conn) (net.Conn, <-chan error) {
	// Create the (buffered) error channel.
	errors := make(chan error, 1)

	// Create the wrapper.
	return &monitoredConnection{
		connection: connection,
		errors:     errors,
	}, errors
}

// Read implements net.Conn.Read.
func (c *monitoredConnection) Read(buffer []byte) (int, error) {
	// Perform the underlying read.
	count, err := c.connection.Read(buffer)

	// If an error occurred, report it.
	if err != nil {
		c.report.Do(func() {
			c.errors <- fmt.Errorf("read error: %w", err)
		})
	}

	// Done.
	return count, err
}

// Write implements net.Conn.Write.
func (c *monitoredConnection) Write(buffer []byte) (int, error) {
	// Perform the underlying write.
	count, err := c.connection.Write(buffer)

	// If an error occurred, report it.
	if err != nil {
		c.report.Do(func() {
			c.errors <- fmt.Errorf("write error: %w", err)
		})
	}

	// Done.
	return count, err
}

// Close implements net.Conn.Close.
func (c *monitoredConnection) Close() error {
	return c.connection.Close()
}

// LocalAddr implements net.Conn.LocalAddr.
func (c *monitoredConnection) LocalAddr() net.Addr {
	return c.connection.LocalAddr()
}

// RemoteAddr implements net.Conn.RemoteAddr.
func (c *monitoredConnection) RemoteAddr() net.Addr {
	return c.connection.RemoteAddr()
}

// SetDeadline implements net.Conn.SetDeadline.
func (c *monitoredConnection) SetDeadline(t time.Time) error {
	return c.connection.SetDeadline(t)
}

// SetReadDeadline implements net.Conn.SetReadDeadline.
func (c *monitoredConnection) SetReadDeadline(t time.Time) error {
	return c.connection.SetReadDeadline(t)
}

// SetWriteDeadline implements net.Conn.SetWriteDeadline.
func (c *monitoredConnection) SetWriteDeadline(t time.Time) error {
	return c.connection.SetWriteDeadline(t)
}
