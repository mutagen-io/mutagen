package webrtcutil

import (
	"bufio"
	"errors"
	"io"
	"math"
	"net"
	"sync"
	"time"

	"github.com/pion/webrtc/v2"

	"github.com/pion/datachannel"
)

const (
	// connectionEstablishmentTimeout is the maximum amount of time that
	// NewConnection will wait for a data channel to connect before timing out.
	connectionEstablishmentTimeout = 10 * time.Second
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
	// dataChannel is the API-level data channel. It is static.
	dataChannel *webrtc.DataChannel
	// callbackOnce ensures only a single invocation of closureCallback. It is
	// static.
	callbackOnce sync.Once
	// closureCallback is a callback that is invoked on explicit closure (i.e. a
	// call to Close, not remote closure). It is static. It may be nil.
	closureCallback func()
	// stateNotifier is used to serialize access to stream, reader, and closed.
	// It is also used to poll/notify on changes to these members, as well as
	// changes to the data channel's buffering level.
	stateNotifier *sync.Cond
	// stream is the detached underlying stream. It will be nil before the
	// connection is open. It is static once set.
	stream datachannel.ReadWriteCloser
	// reader is a buffered wrapper around the stream. It is necessary since
	// the data channel's read buffering is rudimentary and can only return data
	// in message-sized chunks. If a buffer is provided that's less than the
	// message size, the read will fail and the data will be dropped. It will be
	// nil before the connection is open. It is static once set.
	//
	// HACK: We're relying on the implementation details of bufio.Reader's Read
	// method. Specifically, we're assuming that it only ever performs a read on
	// the underlying stream if its buffer is empty (thus guaranteeing the
	// minimum buffer size necessary to receive any WebRTC message). One could
	// argue that this behavior is guaranteed by the following documentation
	// from bufio.Reader.Read: "The bytes are taken from at most one Read on the
	// underlying Reader". Taken literally, it means that bytes from multiple
	// reads won't be mixed into a single output buffer, though more loosely it
	// might be interpreted as meaning that a call to bufio.Reader.Read will
	// perform at most one read (potentially to top up a non-empty buffer). And
	// indeed the bufio.Reader.fill method will mix data from multiple reads in
	// the buffer, though fortunately it isn't called by bufio.Reader.Read. So
	// relying on this behavior is slightly sketchy, but it if this changes we
	// should catch it fairly easily since stream reads will fail with
	// io.ErrShortBuffer. We tolerate this uncertainty because of the stability,
	// error tracking and optimizations (e.g. copy avoidance) that bufio.Reader
	// provides.
	reader *bufio.Reader
	// closed indicates that the underlying stream has been closed, either due
	// to explicit closure, remote closure, or an error. It is static once set.
	closed bool
}

// NewConnection creates a new connection object (that implements net.Conn)
// using the specified data channel as its underlying transport. The data
// channel must support stream detaching (which it will if created from this
// package's API instance), otherwise a panic will occur. If provided, the
// closure callback will be invoked the first time (and only the first time)
// that the connections Close method is called.
func NewConnection(dataChannel *webrtc.DataChannel, closureCallback func()) net.Conn {
	// Create the connection object.
	connection := &connection{
		dataChannel:     dataChannel,
		closureCallback: closureCallback,
		stateNotifier:   sync.NewCond(&sync.Mutex{}),
	}

	// Monitor for establishment of the connection. If the connection is already
	// closed, then we re-invoke close on the data channel
	dataChannel.OnOpen(func() {
		connection.stateNotifier.L.Lock()
		if connection.closed {
			dataChannel.Close()
		} else if connection.stream != nil {
			dataChannel.Close()
			connection.closed = true
		} else if stream, err := dataChannel.Detach(); err != nil {
			dataChannel.Close()
			connection.closed = true
		} else {
			connection.stream = stream
			connection.reader = bufio.NewReaderSize(stream, minimumDataChannelReadBufferSize)
		}
		connection.stateNotifier.Broadcast()
		connection.stateNotifier.L.Unlock()
	})

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
	// ignore the error.
	dataChannel.OnError(func(err error) {
		connection.stateNotifier.L.Lock()
		if !connection.closed {
			dataChannel.Close()
			connection.closed = true
			connection.stateNotifier.Broadcast()
		}
		connection.stateNotifier.L.Unlock()
	})

	// Monitor for closure. If the connection is already closed, then there's
	// nothing we need to do.
	dataChannel.OnClose(func() {
		connection.stateNotifier.L.Lock()
		if !connection.closed {
			connection.closed = true
			connection.stateNotifier.Broadcast()
		}
		connection.stateNotifier.L.Unlock()
	})

	// Done.
	return connection
}

// Read implements net.Conn.Read.
func (c *connection) Read(buffer []byte) (int, error) {
	// Wait until closure or until the reader has been set.
	c.stateNotifier.L.Lock()
	for {
		if c.closed {
			c.stateNotifier.L.Unlock()
			return 0, errors.New("connection closed")
		} else if c.reader != nil {
			break
		}
		c.stateNotifier.Wait()
	}
	c.stateNotifier.L.Unlock()

	// Perform the read using the reader.
	return c.reader.Read(buffer)
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
