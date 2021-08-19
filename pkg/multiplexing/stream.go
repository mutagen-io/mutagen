package multiplexing

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/mutagen-io/mutagen/pkg/multiplexing/ring"
)

var (
	// ErrWriteClosed is returned from operations that fail due to a stream
	// being closed for writing. It is analgous to net.ErrClosed, but indicates
	// that only the write portion of a stream is closed.
	ErrWriteClosed = errors.New("closed for writing")
	// errRemoteClosed is a version of net.ErrorClosed that indicates a stream
	// was closed on the remote.
	errRemoteClosed = fmt.Errorf("remote: %w", net.ErrClosed)
)

// Stream represents a single multiplexed stream. It implements net.Conn but
// also provides a CloseWrite method for half-closures.
type Stream struct {
	// multiplexer is the parent multiplexer.
	multiplexer *Multiplexer
	// identifier is the stream identifier.
	identifier uint64

	// established is closed by the multiplexer if and when the stream is fully
	// established. It may never be closed if the stream is never accepted or is
	// rejected.
	established chan struct{}

	// remoteClosedWrite is closed by the multiplexer's reader Goroutine if and
	// when it receives a close write message for the stream from the remote.
	remoteClosedWrite chan struct{}
	// remoteClosed is closed by the multiplexer's reader Goroutine if and when
	// it receives a close message for the stream from the remote.
	remoteClosed chan struct{}

	// closeOnce guards closure of closed.
	closeOnce sync.Once
	// closed is closed when the stream is closed.
	closed chan struct{}

	// readDeadline holds the timer used to regulate read deadlines. The timer
	// itself is used as a semaphor to serialize read operations. The holder of
	// the timer is responsible for processing deadline set operations on the
	// readDeadlineSet channel if the timer is to be held in a blocking manner.
	// The holder is also responsible for setting the readDeadlineExpired field
	// if the timer is observed to expire.
	readDeadline chan *time.Timer
	// readDeadlineSet is used to signal read deadline set operations to the
	// current holder of the read deadline timer.
	readDeadlineSet chan time.Time
	// readDeadlineExpired is used to record that the holder of the read
	// deadline timer saw it expire.
	readDeadlineExpired bool

	// receiveBufferLock guards access to receiveBuffer and write access to
	// receiveBufferReady.
	receiveBufferLock sync.Mutex
	// receiveBuffer is the inbound data buffer.
	receiveBuffer *ring.Buffer
	// receiveBufferReady is used to signal that receiveBuffer is non-empty.
	// Read access to this channel is guarded by holding the read deadline timer
	// (i.e. being the current reader). Write access is guarded by holding
	// receiveBufferLock. When receiveBufferLock is not held, this channel must
	// be empty if receiveBuffer is empty. Note that this channel may be empty
	// if receiveBuffer is non-empty in the case that a reader has drained it
	// and is now waiting for receiveBufferLock. This channel must be written to
	// by the holder of receiveBufferLock if receiveBuffer transitions from
	// empty to non-empty while the lock is held.
	receiveBufferReady chan struct{}

	// closeWriteOnce guards closure of closedWrite.
	closeWriteOnce sync.Once
	// closedWrite is closed when the stream is closed for writing.
	closedWrite chan struct{}

	// writeDeadline holds the timer used to regulate write deadlines. The timer
	// itself is used as a semaphor to serialize write operations. The holder of
	// the timer is responsible for processing deadline set operations on the
	// writeDeadlineSet channel if the timer is to be held in a blocking manner.
	// The holder is also responsible for setting the writeDeadlineExpired field
	// if the timer is observed to expire.
	writeDeadline chan *time.Timer
	// writeDeadlineSet is used to signal write deadline set operations to the
	// current holder of the write deadline timer.
	writeDeadlineSet chan time.Time
	// readDeadlineExpired is used to record that the holder of the write
	// deadline timer saw it expire.
	writeDeadlineExpired bool

	// sendWindowLock guards access to sendWindow and write access to
	// sendWindowReady.
	sendWindowLock sync.Mutex
	// sendWindow is the current send window.
	sendWindow uint64
	// sendWindowReady is used to signal that sendWindow is non-zero. Read
	// access to this channel is guarded by holding the write deadline timer
	// (i.e. being the current writer). Write access is guarded by holding
	// sendWindowLock. When sendWindowLock is not held, this channel must be
	// empty if sendWindow is zero. Note that this channel may be empty if
	// sendWindow is non-zero in the case that a writer has drained it and is
	// now waiting for sendWindowLock. This channel must be written to by the
	// holder of sendWindowLock if sendWindow transitions from zero to non-zero
	// while the lock is held.
	sendWindowReady chan struct{}
}

// newStoppedTimer creates a new stopped and drained timer.
func newStoppedTimer() *time.Timer {
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	return timer
}

// newStream constructs a new stream.
func newStream(multiplexer *Multiplexer, identifier uint64, receiveWindow int) *Stream {
	// Create the stream.
	stream := &Stream{
		multiplexer:        multiplexer,
		identifier:         identifier,
		established:        make(chan struct{}),
		remoteClosedWrite:  make(chan struct{}),
		remoteClosed:       make(chan struct{}),
		closed:             make(chan struct{}),
		readDeadline:       make(chan *time.Timer, 1),
		readDeadlineSet:    make(chan time.Time),
		receiveBuffer:      ring.NewBuffer(receiveWindow),
		receiveBufferReady: make(chan struct{}, 1),
		closedWrite:        make(chan struct{}),
		writeDeadline:      make(chan *time.Timer, 1),
		writeDeadlineSet:   make(chan time.Time),
		sendWindowReady:    make(chan struct{}, 1),
	}
	stream.readDeadline <- newStoppedTimer()
	stream.writeDeadline <- newStoppedTimer()

	// Done.
	return stream
}

// Read implements net.Conn.Read.
func (s *Stream) Read(buffer []byte) (int, error) {
	// Check for persistent pre-existing error conditions that would prevent a
	// read from succeeding. While we could just allow these to bubble up in the
	// select operations below, their priority in that case would be random,
	// whereas we want error conditions to be returned consistently once they
	// exist. Thus, we cascade these checks in order of reporting priority to
	// ensure consistent error values on subsequent calls once their respective
	// error conditions exist and have been observed for the first time.
	if isClosed(s.closed) {
		return 0, net.ErrClosed
	} else if isClosed(s.multiplexer.closed) {
		return 0, ErrMultiplexerClosed
	}

	// Acquire the read deadline timer, which gives us exclusive read access.
	// It's important to monitor for local stream closure here because that
	// indicates that the read deadline timer has been removed from circulation.
	var readDeadlineTimer *time.Timer
	select {
	case readDeadlineTimer = <-s.readDeadline:
	case <-s.closed:
		return 0, net.ErrClosed
	case <-s.multiplexer.closed:
		return 0, ErrMultiplexerClosed
	}

	// Defer return of the read deadline timer.
	defer func() {
		s.readDeadline <- readDeadlineTimer
	}()

	// Check if the read deadline is already expired.
	if s.readDeadlineExpired {
		return 0, os.ErrDeadlineExceeded
	} else if wasPopulatedWithTime(readDeadlineTimer.C) {
		s.readDeadlineExpired = true
		return 0, os.ErrDeadlineExceeded
	}

	// Wait until the read buffer is populated, the remote cleanly closes the
	// stream, or an error occurs.
	var bufferReady bool
	for !bufferReady {
		select {
		case <-s.receiveBufferReady:
			bufferReady = true
		case <-s.remoteClosedWrite:
			select {
			case <-s.receiveBufferReady:
				bufferReady = true
			default:
				return 0, io.EOF
			}
		case <-s.remoteClosed:
			select {
			case <-s.receiveBufferReady:
				bufferReady = true
			default:
				return 0, io.EOF
			}
		case <-s.closed:
			return 0, net.ErrClosed
		case <-s.multiplexer.closed:
			return 0, ErrMultiplexerClosed
		case <-readDeadlineTimer.C:
			s.readDeadlineExpired = true
			return 0, os.ErrDeadlineExceeded
		case deadline := <-s.readDeadlineSet:
			setStreamDeadline(readDeadlineTimer, &s.readDeadlineExpired, deadline)
			if s.readDeadlineExpired {
				return 0, os.ErrDeadlineExceeded
			}
		}
	}

	// Perform a read from the buffer and ensure that the readiness channel is
	// left in an appropriate state.
	s.receiveBufferLock.Lock()
	count, _ := s.receiveBuffer.Read(buffer)
	if s.receiveBuffer.Used() > 0 {
		s.receiveBufferReady <- struct{}{}
	}
	s.receiveBufferLock.Unlock()

	// Send a window update corresponding to the amount that we read.
	select {
	case s.multiplexer.enqueueWindowIncrement <- windowIncrement{s.identifier, uint64(count)}:
	case <-s.multiplexer.closed:
		return count, ErrMultiplexerClosed
	}

	// Success.
	return count, nil
}

// min returns the lesser of a or b.
func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// Write implements net.Conn.Write.
func (s *Stream) Write(data []byte) (int, error) {
	// Check for persistent pre-existing error conditions that would prevent a
	// write from succeeding. While we could just allow these to bubble up in
	// the select operations below, their priority in that case would be random,
	// whereas we want error conditions to be returned consistently once they
	// exist. Thus, we cascade these checks in order of reporting priority to
	// ensure consistent error values on subsequent calls once their respective
	// error conditions exist and have been observed for the first time.
	if isClosed(s.closed) {
		return 0, net.ErrClosed
	} else if isClosed(s.closedWrite) {
		return 0, ErrWriteClosed
	} else if isClosed(s.multiplexer.closed) {
		return 0, ErrMultiplexerClosed
	} else if isClosed(s.remoteClosed) {
		return 0, errRemoteClosed
	}

	// Acquire the write deadline timer, which gives us exclusive write access.
	// We monitor for the same set of errors as above, though it's particularly
	// important to monitor for local write closure because that indicates that
	// the write deadline timer has been removed from circulation.
	var writeDeadlineTimer *time.Timer
	select {
	case writeDeadlineTimer = <-s.writeDeadline:
	case <-s.closed:
		return 0, net.ErrClosed
	case <-s.closedWrite:
		return 0, ErrWriteClosed
	case <-s.multiplexer.closed:
		return 0, ErrMultiplexerClosed
	case <-s.remoteClosed:
		return 0, errRemoteClosed
	}

	// Defer return of the write deadline timer.
	defer func() {
		s.writeDeadline <- writeDeadlineTimer
	}()

	// Check if the write deadline is already expired.
	if s.writeDeadlineExpired {
		return 0, os.ErrDeadlineExceeded
	} else if wasPopulatedWithTime(writeDeadlineTimer.C) {
		s.writeDeadlineExpired = true
		return 0, os.ErrDeadlineExceeded
	}

	// Loop until all data has been written or an error occurs.
	var count int
	for len(data) > 0 {
		// Loop until we have a combination of non-zero send window and a write
		// buffer to transmit data. We only start polling for a write buffer
		// once we have at least some non-zero amount of send window capacity.
		var haveNonZeroSendWindow bool
		var writeBuffer *messageBuffer
		for writeBuffer == nil {
			// Check if we're polling for the write buffer.
			writeBufferAvailable := s.multiplexer.writeBufferAvailable
			if !haveNonZeroSendWindow {
				writeBufferAvailable = nil
			}

			// Perform polling. If we fail due to deadline expiration while
			// waiting for a write buffer to become available, then we need to
			// resignal send window readiness for future writes, because we will
			// have drained the channel. Any other error condition is terminal,
			// so there's no need to resginal readiness in those cases.
			select {
			case <-s.sendWindowReady:
				haveNonZeroSendWindow = true
			case writeBuffer = <-writeBufferAvailable:
			case <-s.closed:
				return count, net.ErrClosed
			case <-s.closedWrite:
				return count, ErrWriteClosed
			case <-s.multiplexer.closed:
				return count, ErrMultiplexerClosed
			case <-s.remoteClosed:
				return count, errRemoteClosed
			case <-writeDeadlineTimer.C:
				if haveNonZeroSendWindow {
					s.sendWindowLock.Lock()
					s.sendWindowReady <- struct{}{}
					s.sendWindowLock.Unlock()
				}
				s.writeDeadlineExpired = true
				return count, os.ErrDeadlineExceeded
			case deadline := <-s.writeDeadlineSet:
				setStreamDeadline(writeDeadlineTimer, &s.writeDeadlineExpired, deadline)
				if s.writeDeadlineExpired {
					if haveNonZeroSendWindow {
						s.sendWindowLock.Lock()
						s.sendWindowReady <- struct{}{}
						s.sendWindowLock.Unlock()
					}
					return count, os.ErrDeadlineExceeded
				}
			}
		}

		// Compute our write window and ensure the that the readiness channel is
		// left in an appropriate state.
		s.sendWindowLock.Lock()
		window := min(s.sendWindow, min(uint64(len(data)), maximumStreamDataBlockSize))
		s.sendWindow -= window
		if s.sendWindow > 0 {
			s.sendWindowReady <- struct{}{}
		}
		s.sendWindowLock.Unlock()

		// Encode the stream data message and queue it for transmission.
		writeBuffer.encodeStreamDataMessage(s.identifier, data[:window])
		s.multiplexer.writeBufferPending <- writeBuffer

		// Reduce the remaining data slice and update the count.
		data = data[window:]
		count += int(window)
	}

	// Success.
	return count, nil
}

// closeWrite is the internal write closure method. It makes transmission of the
// stream close write message optional.
func (s *Stream) closeWrite(sendCloseWriteMessage bool) (err error) {
	// Perform write closure idempotently.
	s.closeWriteOnce.Do(func() {
		// Signal write closure internally.
		close(s.closedWrite)

		// Wait for all writers to unblock by acquiring the write deadline and
		// taking it out of circulation (and ensuring that it's stopped).
		writeDeadlineTimer := <-s.writeDeadline
		writeDeadlineTimer.Stop()

		// If requested, queue transmission of a close write message.
		if sendCloseWriteMessage {
			select {
			case s.multiplexer.enqueueCloseWrite <- s.identifier:
			case <-s.multiplexer.closed:
				err = ErrMultiplexerClosed
			}
		}
	})

	// Done.
	return
}

// CloseWrite performs half-closure (write-closure) of the stream. Any blocked
// Write or SetWriteDeadline calls will be unblocked. Subsequent calls to
// CloseWrite are no-ops and will return nil.
func (s *Stream) CloseWrite() error {
	return s.closeWrite(true)
}

// close is the internal closure method. It makes transmission of the stream
// close message optional.
func (s *Stream) close(sendCloseMessage bool) (err error) {
	// Terminate writing if it hasn't been terminated already, but don't queue
	// a close write message because we're about to send a full close message.
	s.closeWrite(false)

	// Perform full closure idempotently.
	s.closeOnce.Do(func() {
		// Signal closure internally.
		close(s.closed)

		// Wait for all readers to unblock by acquiring the read deadline and
		// taking it out of circulation (and ensuring that it's stopped).
		// Writers will already have unblocked by the time the closeWrite call
		// above returned.
		readDeadlineTimer := <-s.readDeadline
		readDeadlineTimer.Stop()

		// If requested, queue transmission of a close message.
		if sendCloseMessage {
			select {
			case s.multiplexer.enqueueClose <- s.identifier:
			case <-s.multiplexer.closed:
				err = ErrMultiplexerClosed
			}
		}

		// Deregister the stream from the parent multiplexer.
		s.multiplexer.streamLock.Lock()
		delete(s.multiplexer.streams, s.identifier)
		s.multiplexer.streamLock.Unlock()
	})

	// Done.
	return
}

// Close implements net.Conn.Close. Subsequent calls to Close are no-ops and
// will return nil.
func (s *Stream) Close() error {
	return s.close(true)
}

// LocalAddr implements net.Conn.LocalAddr.
func (s *Stream) LocalAddr() net.Addr {
	return &streamAddress{identifier: s.identifier}
}

// RemoteAddr implements net.Conn.RemoteAddr.
func (s *Stream) RemoteAddr() net.Addr {
	return &streamAddress{remote: true, identifier: s.identifier}
}

// SetDeadline implements net.Conn.SetDeadline.
func (s *Stream) SetDeadline(deadline time.Time) error {
	// Set the read deadline.
	if err := s.SetReadDeadline(deadline); err != nil {
		return fmt.Errorf("unable to set read deadline: %w", err)
	}

	// Set the write deadline.
	if err := s.SetWriteDeadline(deadline); err != nil {
		return fmt.Errorf("unable to set write deadline: %w", err)
	}

	// Success.
	return nil
}

// setStreamDeadline is an internal deadline update function for setting read
// and write deadlines for streams. It must only be called by the holder of the
// respective timer.
func setStreamDeadline(timer *time.Timer, expired *bool, deadline time.Time) {
	// Ensure that the timer is stopped and drained. We don't know its previous
	// state (it may have expired without anyone seeing it or may have been
	// stopped and drained previously), so we perform a non-blocking drain if
	// it's already stopped or expired. We do know that no drain is necessary if
	// the timer is successfully stopped while active, because we never reset a
	// timer without draining it first.
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}

	// Handle the update based on the deadline time.
	if deadline.IsZero() {
		*expired = false
	} else if duration := time.Until(deadline); duration <= 0 {
		*expired = true
	} else {
		timer.Reset(duration)
	}
}

// SetReadDeadline implements net.Conn.SetReadDeadline.
func (s *Stream) SetReadDeadline(deadline time.Time) error {
	// Block until the read deadline is set (by us or its current holder) or
	// until the stream is closed for reading (at which point the read deadline
	// timer is taken out of circulation).
	select {
	case readDeadlineTimer := <-s.readDeadline:
		setStreamDeadline(readDeadlineTimer, &s.readDeadlineExpired, deadline)
		s.readDeadline <- readDeadlineTimer
		return nil
	case s.readDeadlineSet <- deadline:
		return nil
	case <-s.closed:
		return net.ErrClosed
	}
}

// SetWriteDeadline implements net.Conn.SetWriteDeadline.
func (s *Stream) SetWriteDeadline(deadline time.Time) error {
	// Poll until the write deadline is set (by us or its current holder) or
	// until the stream is closed for writing (at which point the write deadline
	// timer is taken out of circulation).
	select {
	case writeDeadlineTimer := <-s.writeDeadline:
		setStreamDeadline(writeDeadlineTimer, &s.writeDeadlineExpired, deadline)
		s.writeDeadline <- writeDeadlineTimer
		return nil
	case s.writeDeadlineSet <- deadline:
		return nil
	case <-s.closedWrite:
		return ErrWriteClosed
	}
}
