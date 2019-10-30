package webrtcutil

import (
	"context"
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
	// connectionEstablishmentTimeout is the maximum amount of time that
	// NewConnection will wait for a data channel to connect before timing out.
	connectionEstablishmentTimeout = 10 * time.Second
	// minimumDataChannelReadBufferSize is the minimum size required for buffers
	// used in data channel read operations. The underlying SCTP implementation
	// only has rudimentary internal buffering and thus requires that read
	// operations be able to consume the entirety of the next "chunk" in a
	// single read operation.
	// HACK: This number relies on knowledge of pion/sctp's implementation.
	// TODO: Is this value correct? Should we use a margin of 2x on here?
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
	// dataChannel is the API-level data channel.
	dataChannel *webrtc.DataChannel
	// stream is the detached underlying stream. It will be nil before the
	// connection is open.
	stream datachannel.ReadWriteCloser
	// readPipeWriter is the incoming data pipe writer.
	readPipeWriter *io.PipeWriter
	// readPipeReader is the incoming data pipe reader.
	readPipeReader *io.PipeReader
	// writeNotifier is a condition variable used to regulate writes. It is used
	// to track write buffer threshold changes and to track closure.
	writeNotifier *sync.Cond
	// writeClosed indicates whether or not the connection is closed to writes.
	writeClosed bool
}

// NewConnection creates a new net.Conn using the specified data channel as its
// underlying transport. The data channel must support stream detaching (which
// it will if created from this package's API instance), otherwise a panic will
// occur. If this function returns an error, then the data channel should be
// considered errored, and it will be the responsibility of the caller to close
// the data channel. If this function succeeds, then the data channel should not
// be closed directly but will instead be closed by closing the returned
// connection.
func NewConnection(dataChannel *webrtc.DataChannel) (net.Conn, error) {
	// Monitor for the first data channel open event. Later events may be
	// dropped or ignored.
	dataChannelOpens := make(chan struct{}, 1)
	dataChannel.OnOpen(func() {
		select {
		case dataChannelOpens <- struct{}{}:
		default:
		}
	})

	// Monitor for the first data channel error. Later errors may be dropped or
	// ignored.
	dataChannelErrors := make(chan error, 1)
	dataChannel.OnError(func(err error) {
		select {
		case dataChannelErrors <- err:
		default:
		}
	})
	dataChannel.OnClose(func() {
		select {
		case dataChannelErrors <- errors.New("data channel closed unexpectedly"):
		default:
		}
	})

	// Create a timeout context to regulate connection establishment and ensure
	// that it's cancelled.
	timeout, cancelTimeout := context.WithTimeout(context.Background(), connectionEstablishmentTimeout)
	defer cancelTimeout()

	// Wait for connection establishment, an error, or a timeout.
	select {
	case <-dataChannelOpens:
	case err := <-dataChannelErrors:
		return nil, fmt.Errorf("data channel error: %w", err)
	case <-timeout.Done():
		return nil, errors.New("connection timeout")
	}

	// Detach the underlying stream.
	stream, err := dataChannel.Detach()
	if err != nil {
		return nil, fmt.Errorf("unable to detach SCTP stream")
	}

	// Create a pipe to pass data from the read loop to readers.
	// HACK: We rely on the fact that io.Pipe provides unblock-on-Close
	// behavior, which isn't documented. If it ever stops implementing this
	// behavior, we could switch to net.Pipe, which is guaranteed to provide
	// this behavior since it supports net.Conn, but it uses channels internally
	// (to support deadlines) whereas io.Pipe uses more performant mutexes and
	// condition variables.
	readPipeReader, readPipeWriter := io.Pipe()

	// Create a write notifier condition variable and have it trigger when the
	// buffer write data size drops below a tolerable threshold. It's important
	// to note that we have to hold the lock on the condition variable when
	// broadcasting (even though the sync package doesn't require this) to
	// ensure that no writer is inside the condition checking block but not yet
	// in a Wait call, otherwise our broadcast can go unnoticed and the writer
	// can become stuck in a Wait.
	writeNotifier := sync.NewCond(&sync.Mutex{})
	dataChannel.SetBufferedAmountLowThreshold(maximumWriteBufferSize)
	dataChannel.OnBufferedAmountLow(func() {
		writeNotifier.L.Lock()
		writeNotifier.Broadcast()
		writeNotifier.L.Unlock()
	})

	// Create the connection object.
	connection := &connection{
		dataChannel:    dataChannel,
		stream:         stream,
		readPipeWriter: readPipeWriter,
		readPipeReader: readPipeReader,
		writeNotifier:  writeNotifier,
	}

	// Start the read loop.
	go connection.readLoop()

	// Success.
	return connection, nil
}

// readLoop is the read loop implementation. It is a necessary workaround for
// data channels' rudimentary internal buffering that only allows data to be
// read out in chunks corresponding to their original transmission size (a
// reasonable design decision that arises from SCTP's message-based nature). If
// a buffer that's too small to read a message is provided, the read will fail
// with io.ErrShortBuffer and the stream will be corrupted. By using a read loop
// with a sufficiently sized buffer and an io.Pipe to adapt buffer sizes, we can
// allow read sizes that don't correspond to message sizes.
func (c *connection) readLoop() {
	// Create the read buffer.
	buffer := make([]byte, minimumDataChannelReadBufferSize)

	// Loop and read data, watching for closure of the underlying connection or
	// pipe.
	for {
		// Perform a read. This will unblock if the data channel is closed.
		read, text, err := c.stream.ReadDataChannel(buffer)
		if err != nil {
			c.readPipeWriter.CloseWithError(err)
			return
		}

		// Ensure that data type is correct.
		if text {
			c.readPipeWriter.CloseWithError(errors.New("text data received"))
			return
		}

		// Forward the data. This will unblock if the pipe writer is closed.
		if forwarded, err := c.readPipeWriter.Write(buffer[:read]); err != nil {
			c.readPipeWriter.CloseWithError(fmt.Errorf("unable to forward data: %w", err))
			return
		} else if forwarded != read {
			c.readPipeWriter.CloseWithError(io.ErrShortWrite)
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
	// Loop until all data is written or an error occurred.
	var count int
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

		// Wait until the stream buffer is at or below the size threshold.
		// NOTE: Technically this allows us to write up to maximumWriteSize
		// past the buffer threshold, which we may want to fix, though it's fine
		// for now since maximumWriteSize is sufficiently small. Unfortunately
		// the only notification available is when the threshold is crossed (or
		// hit, which is why we have to test for equality here too), so we'd
		// have to poll periodically once at or below the threshold if we wanted
		// to find out when the buffered amount was small enough to avoid
		// exceeding maximumWriteSize.
		c.writeNotifier.L.Lock()
		for {
			if c.writeClosed {
				c.writeNotifier.L.Unlock()
				return count, errors.New("connection closed")
			} else if c.dataChannel.BufferedAmount() <= maximumWriteBufferSize {
				break
			} else {
				c.writeNotifier.Wait()
			}
		}
		c.writeNotifier.L.Unlock()

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
	// Close the incoming data pipe to ensure that neither the read loop nor any
	// readers are blocked on it. This never returns an error.
	c.readPipeWriter.CloseWithError(nil)

	// Unblock and prevent any writes.
	c.writeNotifier.L.Lock()
	c.writeClosed = true
	c.writeNotifier.Broadcast()
	c.writeNotifier.L.Unlock()

	// Close the underlying data channel.
	return c.dataChannel.Close()
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
