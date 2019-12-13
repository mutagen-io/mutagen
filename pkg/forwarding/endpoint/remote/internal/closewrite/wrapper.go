package closewrite

import (
	"encoding/binary"
	"io"
	"math"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// writeBufferPool is a pool for creating and storing write buffers.
var writeBufferPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 2+math.MaxUint16)
	},
}

// closeWriterConnection is a wrapper around a net.Conn that implements a
// CloseWrite method by adding a lightweight framing protocol on top of the
// underlying connection.
type closeWriterConnection struct {
	// connection is the underlying connection.
	connection net.Conn
	// readLock locks read operations on the connection, as well as the
	// previousReadError, eof, and incoming fields.
	readLock sync.Mutex
	// previousReadError stores any previous read error.
	previousReadError error
	// eof indicates whether or not an EOF indicator has been seen.
	eof bool
	// incoming indicates how much data is expected before the next header.
	incoming uint16
	// writeLock locks write operations on the connection, as well as the
	// previousWriteError and writeClosed fields.
	writeLock sync.Mutex
	// previousWriteError stores any previous write error.
	previousWriteError error
	// writeClosed indicates whether or not the connection is closed for writes.
	writeClosed bool
}

// Enable enables CloseWrite for connections that doesn't support it natively.
// It will also work for connections that do support CloseWrite natively, but
// this implementation comes at the cost of some throughput, some additional
// memory usage, and the inability to use deadlines, so it should only be used
// when absolutely necessary. Most importantly, because this wrapping implements
// a light framing protocol on top of the connection, it must be used on both
// ends of the connection.
func Enable(connection net.Conn) net.Conn {
	return &closeWriterConnection{
		connection: connection,
	}
}

// Read implements net.Conn.Read.
func (c *closeWriterConnection) Read(buffer []byte) (int, error) {
	// Acquire the read lock and defer its release.
	c.readLock.Lock()
	defer c.readLock.Unlock()

	// Check for any previous read error.
	if c.previousReadError != nil {
		return 0, errors.Wrap(c.previousReadError, "previous read error")
	}

	// Check if we've already seen EOF. If so, we're done.
	if c.eof {
		return 0, io.EOF
	}

	// If we're expecting another header, then attempt to read it.
	if c.incoming == 0 {
		// Perform the read.
		var header [2]byte
		if _, err := io.ReadFull(c.connection, header[:]); err != nil {
			c.previousReadError = errors.Wrap(err, "unable to read header")
			return 0, c.previousReadError
		}

		// Decode the header.
		c.incoming = binary.BigEndian.Uint16(header[:])

		// Watch for an EOF indicator.
		if c.incoming == 0 {
			c.eof = true
			return 0, io.EOF
		}
	}

	// If the buffer has a length of 0, then we can bail at this point.
	if len(buffer) == 0 {
		return 0, nil
	}

	// Determine how much we want to read. We'll only allow as many bytes as we
	// expect before the next header, because trying to manage reads across
	// multiple headers is too complex.
	desired := len(buffer)
	if incoming := int(c.incoming); incoming < desired {
		desired = incoming
	}

	// Perform the read and track any errors.
	count, err := io.ReadFull(c.connection, buffer[:desired])
	if err != nil {
		c.previousReadError = err
	}
	c.incoming -= uint16(count)

	// Done.
	return count, err
}

// Write implements net.Conn.Write.
func (c *closeWriterConnection) Write(data []byte) (int, error) {
	// Acquire the write lock and defer its release.
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	// Check for any previous write error.
	if c.previousWriteError != nil {
		return 0, errors.Wrap(c.previousWriteError, "previous write error")
	}

	// Ensure that the connection isn't closed for writes.
	if c.writeClosed {
		return 0, errors.New("closed for writes")
	}

	// Treat 0-length writes as a no-op since we use 0 headers to indicate EOF.
	if len(data) == 0 {
		return 0, nil
	}

	// Disallow writes larger than our framing header allows.
	if len(data) > math.MaxUint16 {
		return 0, errors.New("data too large to frame")
	}

	// Grab a buffer to compose the write and defer its return to the pool.
	buffer := writeBufferPool.Get().([]byte)
	defer writeBufferPool.Put(buffer)

	// Write the header.
	binary.BigEndian.PutUint16(buffer[:2], uint16(len(data)))

	// Write the data.
	copy(buffer[2:], data)

	// Write the data.
	count, err := c.connection.Write(buffer[:2+len(data)])
	if err != nil {
		if count < 2 {
			c.previousWriteError = errors.Wrap(err, "unable to write header")
			return 0, c.previousWriteError
		} else {
			c.previousWriteError = err
			return count - 2, err
		}
	}

	// Success.
	return count - 2, nil
}

// CloseWrite implements CloseWriter.CloseWrite.
func (c *closeWriterConnection) CloseWrite() error {
	// Acquire the write lock and defer its release.
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	// Check for any previous write error.
	if c.previousWriteError != nil {
		return errors.Wrap(c.previousWriteError, "previous write error")
	}

	// Ensure that the connection isn't already closed for writes.
	if c.writeClosed {
		return errors.New("already closed for writes")
	}

	// Mark the connection as closed for writes.
	c.writeClosed = true

	// Write an all-0 header to indicate EOF.
	var header [2]byte
	if _, err := c.connection.Write(header[:]); err != nil {
		c.previousWriteError = errors.Wrap(err, "unable to write EOF header")
		return c.previousWriteError
	}

	// Success.
	return nil
}

// Close implements net.Conn.Close.
func (c *closeWriterConnection) Close() error {
	return c.connection.Close()
}

// LocalAddr implements net.Conn.LocalAddr.
func (c *closeWriterConnection) LocalAddr() net.Addr {
	return c.connection.LocalAddr()
}

// RemoteAddr implements net.Conn.RemoteAddr.
func (c *closeWriterConnection) RemoteAddr() net.Addr {
	return c.connection.RemoteAddr()
}

// SetDeadline implements net.Conn.SetDeadline.
func (c *closeWriterConnection) SetDeadline(t time.Time) error {
	return errors.New("deadlines not supported by write closer connections")
}

// SetReadDeadline implements net.Conn.SetReadDeadline.
func (c *closeWriterConnection) SetReadDeadline(t time.Time) error {
	return errors.New("read deadlines not supported by write closer connections")
}

// SetWriteDeadline implements net.Conn.SetWriteDeadline.
func (c *closeWriterConnection) SetWriteDeadline(t time.Time) error {
	return errors.New("write deadlines not supported by write closer connections")
}
