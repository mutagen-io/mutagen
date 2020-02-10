package webrtcutil

import (
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"

	"github.com/pion/webrtc/v2"

	"github.com/pion/datachannel"
)

const (
	// minimumDataChannelReadBufferSize is the minimum size required for buffers
	// used in data channel read operations. The underlying SCTP implementation
	// only has rudimentary internal buffering and thus requires that read
	// operations be able to consume the entirety of the next "chunk" in a
	// single read operation.
	// HACK: This number relies on knowledge of pion/sctp's implementation.
	minimumDataChannelReadBufferSize = math.MaxUint16
	// maximumWriteSize is the maximum buffer size that the WebRTC packages will
	// accept in a single write.
	// HACK: This number relies on knowledge of pion/sctp's implementation.
	maximumWriteSize = math.MaxUint16
	// maximumWriteBufferSize is the maximum amount of data that a connection
	// will allow to be buffered for writes before blocking.
	// TODO: This number is pulled out of thin air and can almost certainly be
	// optimized. Since SCTP doesn't have its own buffer backpressure with nice
	// BDP estimation, we choose a buffer size that should account for long fat
	// networks, but it may be too big.
	maximumWriteBufferSize = 1024 * 1024
)

// address implements net.Addr for data channel connections.
type address struct{}

// Network returns the connection protocol name.
func (_ address) Network() string {
	// TODO: If we know the underlying protocol for the peer connection, e.g.
	// STUN or TURN, we could probably return the relevant IANA scheme.
	return "webrtc"
}

// String returns the connection address.
func (_ address) String() string {
	// TODO: We could incorporate the data channel ID here.
	return "data channel"
}

// connection is a wrapper around Pion's DataChannel type that implements a
// net.Conn-compatible interface, namely one that is stream-based, doesn't
// restrict read and write sizes, unblocks Read/Write on close, and properly
// enforces backpressure.
type connection struct {
	// readPipeWriter is the write end of the read pipe. It is static.
	readPipeWriter *io.PipeWriter
	// readPipeReader is the read end of the read pipe. The pipe is never closed
	// from this end. It is static.
	readPipeReader io.Reader
	// callbackOnce ensures only a single invocation of closureCallback. It is
	// static.
	callbackOnce sync.Once
	// closureCallback is a callback that is invoked on explicit closure (i.e. a
	// call to Close, not remote closure). It is static. It may be nil.
	closureCallback func()
	// stateNotifier is used to serialize access to dataChannel, stream, and
	// closed, as well as the pipe writer's CloseWithError method. It is also
	// used to poll/notify on changes to these members, including the data
	// channel's buffering level.
	stateNotifier *sync.Cond
	// dataChannel is the API-level data channel. It is static.
	dataChannel *webrtc.DataChannel
	// stream is the detached underlying stream. It will be nil before the
	// connection is open. It is static once set.
	stream datachannel.ReadWriteCloser
	// closed indicates that the underlying stream has been closed, either due
	// to explicit closure, remote closure, or an error. It is static once set
	// to true.
	closed bool
}

// NewConnection creates a new connection object (that implements net.Conn)
// using the specified data channel as its underlying transport. The data
// channel must support stream detaching (which it will if created from this
// package's API instance), otherwise a panic will occur. If provided, the
// closure callback will be invoked the first time (and only the first time)
// that the connections Close method is called.
func NewConnection(dataChannel *webrtc.DataChannel, closureCallback func()) net.Conn {
	// Create the read pipe.
	readPipeReader, readPipeWriter := io.Pipe()

	// Create the connection object.
	connection := &connection{
		readPipeWriter:  readPipeWriter,
		readPipeReader:  readPipeReader,
		closureCallback: closureCallback,
		stateNotifier:   sync.NewCond(&sync.Mutex{}),
		dataChannel:     dataChannel,
	}

	// Start the read loop.
	go connection.readLoop()

	// Set the low buffer threshold and wire up the associated callback to
	// trigger a state change notification. We only trigger a notification if
	// the connection isn't already closed.
	dataChannel.SetBufferedAmountLowThreshold(maximumWriteBufferSize)
	dataChannel.OnBufferedAmountLow(func() {
		connection.stateNotifier.L.Lock()
		if !connection.closed {
			connection.stateNotifier.Broadcast()
		}
		connection.stateNotifier.L.Unlock()
	})

	// Monitor for errors. If the connection is already closed, then we just
	// ignore the error. In practice, this callback is unnecessary because the
	// data channel doesn't yield error events when detached, but we need to
	// handle it anyway in case the implementation changes.
	dataChannel.OnError(func(err error) {
		connection.stateNotifier.L.Lock()
		if !connection.closed {
			dataChannel.Close()
			readPipeWriter.CloseWithError(err)
			connection.closed = true
			connection.stateNotifier.Broadcast()
		}
		connection.stateNotifier.L.Unlock()
	})

	// Monitor for closure. If the connection is already closed, then there's
	// nothing we need to do. In practice, this callback is unnecessary because
	// the data channel doesn't yield close events when detached, but we need to
	// handle it anyway in case the implementation changes.
	dataChannel.OnClose(func() {
		connection.stateNotifier.L.Lock()
		if !connection.closed {
			readPipeWriter.CloseWithError(nil)
			connection.closed = true
			connection.stateNotifier.Broadcast()
		}
		connection.stateNotifier.L.Unlock()
	})

	// Monitor for establishment of the connection. We also handle an assortment
	// of cases that shouldn't arise but still need handling.
	dataChannel.OnOpen(func() {
		connection.stateNotifier.L.Lock()
		if connection.closed {
			dataChannel.Close()
		} else if connection.stream != nil {
			dataChannel.Close()
			readPipeWriter.CloseWithError(errors.New("data channel re-opened"))
			connection.closed = true
			connection.stateNotifier.Broadcast()
		} else if stream, err := dataChannel.Detach(); err != nil {
			dataChannel.Close()
			readPipeWriter.CloseWithError(fmt.Errorf("unable to detatch stream: %w", err))
			connection.closed = true
			connection.stateNotifier.Broadcast()
		} else {
			connection.stream = stream
			connection.stateNotifier.Broadcast()
		}
		connection.stateNotifier.L.Unlock()
	})

	// Done.
	return connection
}

// readLoop is the read loop implementation.
//
// This loop is a necessary workaround for data channels' rudimentary internal
// buffering, which only allows data to be read out in chunks corresponding to
// their original transmission size. If a buffer that's too small to read a
// message is provided, the read will fail with io.ErrShortBuffer and the stream
// will be corrupted. By using a read loop with a sufficiently large buffer and
// an io.Pipe to adapt buffer sizes, we can allow read sizes that don't
// correspond to message sizes. In theory, a bufio.Reader with a buffer of the
// required size could also work, but that relies on an implementation detail of
// bufio.Reader.Read (specifically that it only performs a read when its buffer
// is empty), which is only guaranteed in a very strict interpretation of its
// documentation and is violated by other methods on bufio.Reader.
//
// In any case, this loop is equally necessary because close and error events
// aren't delivered by the data channel when running in detached mode. Those
// events are normally generated by the read loop used to handle callback-based
// operation with data channels. Thus, we've effectively replaced that loop with
// one that has better buffering and way fewer allocations.
func (c *connection) readLoop() {
	// Wait until closure until the connection has been established.
	c.stateNotifier.L.Lock()
	for {
		if c.closed {
			c.stateNotifier.L.Unlock()
			return
		} else if c.stream != nil {
			break
		}
		c.stateNotifier.Wait()
	}
	c.stateNotifier.L.Unlock()

	// Create the read buffer.
	buffer := make([]byte, minimumDataChannelReadBufferSize)

	// Loop and forward data.
	for {
		// Perform a read. This will unblock if the data channel is closed.
		read, text, err := c.stream.ReadDataChannel(buffer)

		// Treat text data as an error.
		if err == nil && text {
			err = errors.New("text data received")
		}

		// Handle errors.
		if err != nil {
			c.stateNotifier.L.Lock()
			if !c.closed {
				c.dataChannel.Close()
				c.readPipeWriter.CloseWithError(err)
				c.closed = true
				c.stateNotifier.Broadcast()
			}
			c.stateNotifier.L.Unlock()
			return
		}

		// Forward the data. We rely on the fact that Write won't return an
		// error unless the read end of the pipe is closed (which we don't
		// allow) or the write end of the pipe is closed (in which case we can
		// assume that an error has occurred and the connection is closed). We
		// also rely on the fact that Write can be preempted by closing the
		// write end of the pipe, which isn't fully guaranteed by the io
		// package's pipe implementation, so we include a unit test to verify
		// that behavior.
		if _, err := c.readPipeWriter.Write(buffer[:read]); err != nil {
			return
		}
	}
}

// Read implements net.Conn.Read.
func (c *connection) Read(buffer []byte) (int, error) {
	return c.readPipeReader.Read(buffer)
}

// Write implements net.Conn.Write.
func (c *connection) Write(data []byte) (int, error) {
	// Track how much data we've written.
	var count int

	// Loop until all data is written or an error occurred.
	for len(data) > 0 {
		// Extract the slice to write.
		var partial []byte
		if len(data) > maximumWriteSize {
			partial = data[:maximumWriteSize]
			data = data[maximumWriteSize:]
		} else {
			partial = data
			data = nil
		}

		// Wait until closure or until the stream is set and the data channel
		// buffer is below the size threshold. Technically the check we use here
		// allows us to write up to maximumWriteSize bytes past the buffer
		// threshold, which we may want to fix, though it's fine for now since
		// maximumWriteSize is sufficiently small. Unfortunately, the only
		// notification available is when the threshold is crossed (or hit,
		// which is why we have to test for equality here too), so we'd have to
		// poll periodically once at or below the threshold if we wanted to find
		// out when the buffered amount was small enough to avoid going beyond
		// the limit.
		c.stateNotifier.L.Lock()
		for {
			if c.closed {
				c.stateNotifier.L.Unlock()
				return count, errors.New("connection closed")
			} else if c.stream != nil && c.dataChannel.BufferedAmount() <= maximumWriteBufferSize {
				break
			}
			c.stateNotifier.Wait()
		}
		c.stateNotifier.L.Unlock()

		// Perform the write.
		n, err := c.stream.Write(partial)
		count += n
		if err != nil {
			return count, err
		} else if n != len(partial) {
			return count, io.ErrShortWrite
		}
	}

	// Success.
	return count, nil
}

// Close implements net.Conn.Close.
func (c *connection) Close() error {
	// Perform closure if necessary.
	var result error
	c.stateNotifier.L.Lock()
	if !c.closed {
		result = c.dataChannel.Close()
		c.readPipeWriter.CloseWithError(nil)
		c.closed = true
		c.stateNotifier.Broadcast()
	}
	c.stateNotifier.L.Unlock()

	// Call the closure callback, in any.
	if c.closureCallback != nil {
		c.callbackOnce.Do(c.closureCallback)
	}

	// Done.
	return result
}

// LocalAddr implements net.Conn.LocalAddr.
func (c *connection) LocalAddr() net.Addr {
	return address{}
}

// RemoteAddr implements net.Conn.RemoteAddr.
func (c *connection) RemoteAddr() net.Addr {
	return address{}
}

// SetDeadline implements net.Conn.SetDeadline.
func (c *connection) SetDeadline(_ time.Time) error {
	return errors.New("deadlines not supported by data channel connections")
}

// SetReadDeadline implements net.Conn.SetReadDeadline.
func (c *connection) SetReadDeadline(_ time.Time) error {
	return errors.New("read deadlines not supported by data channel connections")
}

// SetWriteDeadline implements net.Conn.SetWriteDeadline.
func (c *connection) SetWriteDeadline(_ time.Time) error {
	return errors.New("write deadlines not supported by data channel connections")
}
