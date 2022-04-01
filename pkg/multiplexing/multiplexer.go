package multiplexing

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"

	"github.com/mutagen-io/mutagen/pkg/multiplexing/ring"
)

var (
	// ErrMultiplexerClosed is returned from operations that fail due to a
	// multiplexer being closed.
	ErrMultiplexerClosed = errors.New("multiplexer closed")
	// ErrStreamRejected is returned from open operations that fail due to the
	// remote endpoint rejecting the open request.
	ErrStreamRejected = errors.New("stream rejected")
)

// windowIncrement is used to pass a window increment from a stream to the
// multiplexer.
type windowIncrement struct {
	// stream is the stream identifier.
	stream uint64
	// amount is the increment amount.
	amount uint64
}

// Multiplexer provides bidirectional stream multiplexing.
type Multiplexer struct {
	// even indicates whether or not the multiplexer uses even-numbered outbound
	// stream identifiers.
	even bool
	// configuration is the multiplexer configuration.
	configuration *Configuration

	// closeOnce guards closure of closer and closed.
	closeOnce sync.Once
	// closer closes the underlying carrier.
	closer io.Closer
	// closed is closed when the underlying carrier is closed.
	closed chan struct{}
	// internalErrorLock guards access to internalError.
	internalErrorLock sync.RWMutex
	// internalError records the error associated with closure, if any.
	internalError error

	// streamLock guards nextOutboundStreamIdentifier and streams.
	streamLock sync.Mutex
	// nextOutboundStreamIdentifier is the next outbound stream identifier that
	// will be used. It is set to 0 when outbound identifiers are exhausted.
	nextOutboundStreamIdentifier uint64
	// streams maps stream identifiers to their corresponding local stream
	// objects. Stream objects perform their own deregistration when closed.
	streams map[uint64]*Stream
	// pendingInboundStreamIdentifiers is the backlog of pending inbound stream
	// identifiers waiting to be accepted. It is written to only by the reader
	// Goroutine. It has a capacity equal to the accept backlog size.
	pendingInboundStreamIdentifiers chan uint64

	// writeBufferAvailable is the channel where empty outbound message buffers
	// are stored. If a buffer is in this channel, it is guaranteed to have
	// sufficient free space to buffer any single message. Pollers on this
	// channel should always poll on closed simultaneously and terminate if
	// closed is closed.
	writeBufferAvailable chan *messageBuffer
	// writeBufferPending is the channel where non-empty outbound message
	// buffers should be placed to enqueue them for transmission. Writes to this
	// channel are only allowed by holders of outbound message buffers and are
	// guaranteed never to block.
	writeBufferPending chan *messageBuffer

	// enqueueWindowIncrement enqueues transmission of a stream receive window
	// increment message. The amount will be added to any pending window
	// increment. This channel is unbuffered, but guaranteed to be approximately
	// non-blocking as long as the multiplexer is not closed (as indicated by
	// the closed channel).
	enqueueWindowIncrement chan windowIncrement
	// enqueueCloseWrite enqueues transmission of a stream close write message.
	// It should be provided with the stream identifier. This channel is
	// unbuffered, but guaranteed to be approximately non-blocking as long as
	// the multiplexer is not closed (as indicated by the closed channel).
	enqueueCloseWrite chan uint64
	// enqueueClose enqueues transmission of a stream close message. It should
	// be provided with the stream identifier. Any pending window increment or
	// close write messages will be cancelled. This channel is unbuffered, but
	// guaranteed to be approximately non-blocking as long as the multiplexer is
	// not closed (as indicated by the closed channel).
	enqueueClose chan uint64
}

// Multiplex creates a new multiplexer on top of an existing carrier stream. The
// multiplexer takes ownership of the carrier, so it should not be used directly
// after being passed to this function.
//
// Multiplexers are symmetric, meaning that a multiplexer at either end of the
// carrier can both open and accept connections. However, a single asymmetric
// parameter is required to avoid the need for negotiating stream identifiers,
// so the even parameter must be set to true on one endpoint and false on the
// other (using some implicit or out-of-band coordination mechanism, such as
// false for client and true for server). The value of even has no observable
// effect on the multiplexer, other than determining the evenness of outbound
// stream identifiers.
//
// If configuration is nil, the default configuration will be used.
func Multiplex(carrier Carrier, even bool, configuration *Configuration) *Multiplexer {
	// If no configuration was provided, then use default values, otherwise
	// normalize any out-of-range values provided by the caller.
	if configuration == nil {
		configuration = DefaultConfiguration()
	} else {
		configuration.normalize()
	}

	// Create the multiplexer.
	multiplexer := &Multiplexer{
		even:                            even,
		configuration:                   configuration,
		closer:                          carrier,
		closed:                          make(chan struct{}),
		streams:                         make(map[uint64]*Stream),
		pendingInboundStreamIdentifiers: make(chan uint64, configuration.AcceptBacklog),
		writeBufferAvailable:            make(chan *messageBuffer, configuration.WriteBufferCount),
		writeBufferPending:              make(chan *messageBuffer, configuration.WriteBufferCount),
		enqueueWindowIncrement:          make(chan windowIncrement),
		enqueueCloseWrite:               make(chan uint64),
		enqueueClose:                    make(chan uint64),
	}
	if even {
		multiplexer.nextOutboundStreamIdentifier = 2
	} else {
		multiplexer.nextOutboundStreamIdentifier = 1
	}
	for i := 0; i < configuration.WriteBufferCount; i++ {
		multiplexer.writeBufferAvailable <- newMessageBuffer()
	}

	// Start the multiplexer's background Goroutines.
	go multiplexer.run(carrier)

	// Done.
	return multiplexer
}

// run is the primary entry point for the multiplexer's background Goroutines.
func (m *Multiplexer) run(carrier Carrier) {
	// Start the reader Goroutine and monitor for its termination.
	heartbeats := make(chan struct{}, 1)
	readErrors := make(chan error, 1)
	go func() {
		readErrors <- m.read(carrier, heartbeats)
	}()

	// Start the writer Goroutine and monitor for its termination.
	writeErrors := make(chan error, 1)
	go func() {
		writeErrors <- m.write(carrier)
	}()

	// Start the state accumulation/transmission Goroutine. It will only
	// terminate when the multiplexer is closed.
	go m.enqueue()

	// Create a timer to enforce heartbeat reception and defer its shutdown. If
	// inbound heartbeats are not required, then just leave the timer stopped.
	heartbeatTimeout := time.NewTimer(m.configuration.MaximumHeartbeatReceiveInterval)
	if m.configuration.MaximumHeartbeatReceiveInterval > 0 {
		defer heartbeatTimeout.Stop()
	} else {
		if !heartbeatTimeout.Stop() {
			<-heartbeatTimeout.C
		}
	}

	// Loop until failure or multiplexer closure.
	for {
		select {
		case <-heartbeats:
			if m.configuration.MaximumHeartbeatReceiveInterval > 0 {
				if !heartbeatTimeout.Stop() {
					<-heartbeatTimeout.C
				}
				heartbeatTimeout.Reset(m.configuration.MaximumHeartbeatReceiveInterval)
			}
		case err := <-readErrors:
			m.closeWithError(fmt.Errorf("read error: %w", err))
			return
		case err := <-writeErrors:
			m.closeWithError(fmt.Errorf("write error: %w", err))
			return
		case <-heartbeatTimeout.C:
			m.closeWithError(errors.New("heartbeat timeout"))
			return
		case <-m.closed:
			return
		}
	}
}

// read is the entry point for the reader Goroutine.
func (m *Multiplexer) read(reader Carrier, heartbeats chan<- struct{}) error {
	// Create a buffer for reading stream data lengths, which are encoded as
	// 16-bit unsigned integers.
	var lengthBuffer [2]byte

	// Track the range of stream identifiers used by the remote.
	var largestOpenedInboundStreamIdentifier uint64

	// Loop until failure or multiplexure closure.
	for {
		// Read the next message type.
		var kind messageKind
		if k, err := reader.ReadByte(); err != nil {
			return fmt.Errorf("unable to read message kind: %w", err)
		} else {
			kind = messageKind(k)
		}

		// Ensure that the message kind is valid.
		if kind > messageKindStreamClose {
			return fmt.Errorf("received unknown message kind: %#02x", kind)
		}

		// If this is a multiplexer heartbeat message, then strobe the heartbeat
		// channel and continue to the next message.
		if kind == messageKindMultiplexerHeartbeat {
			select {
			case heartbeats <- struct{}{}:
			default:
			}
			continue
		}

		// At this point, we know that this is a stream message, so decode the
		// stream identifier and perform basic validation.
		streamIdentifier, err := binary.ReadUvarint(reader)
		if err != nil {
			return fmt.Errorf("unable to read stream identifier (message kind %#02x): %w", kind, err)
		} else if streamIdentifier == 0 {
			return fmt.Errorf("zero-value stream identifier received (message kind %#02x)", kind)
		}

		// Verify that the stream identifier falls with an acceptable range,
		// depending on its origin and the message kind, and look up the
		// corresponding stream object, if applicable.
		streamIdentifierIsOutbound := m.even == (streamIdentifier%2 == 0)
		var stream *Stream
		if kind == messageKindStreamOpen {
			if streamIdentifierIsOutbound {
				return errors.New("outbound stream identifier used by remote to open stream")
			} else if streamIdentifier <= largestOpenedInboundStreamIdentifier {
				return errors.New("remote stream identifiers not monotonically increasing")
			}
			largestOpenedInboundStreamIdentifier = streamIdentifier
		} else if kind == messageKindStreamAccept && !streamIdentifierIsOutbound {
			return errors.New("inbound stream identifier used by remote to accept stream")
		} else {
			inboundStreamIdentifierOutOfRange := !streamIdentifierIsOutbound &&
				streamIdentifier > largestOpenedInboundStreamIdentifier
			if inboundStreamIdentifierOutOfRange {
				return fmt.Errorf("message (%#02x) received for unopened inbound stream identifier", kind)
			}
			m.streamLock.Lock()
			outboundStreamIdentifierOutOfRange := streamIdentifierIsOutbound &&
				m.nextOutboundStreamIdentifier != 0 &&
				streamIdentifier >= m.nextOutboundStreamIdentifier
			if outboundStreamIdentifierOutOfRange {
				m.streamLock.Unlock()
				return fmt.Errorf("message (%#02x) received for unused outbound stream identifier", kind)
			}
			stream = m.streams[streamIdentifier]
			m.streamLock.Unlock()
		}

		// Handle the remainder of the message based on kind.
		if kind == messageKindStreamOpen {
			// Decode the remote's initial receive window size.
			windowSize, err := binary.ReadUvarint(reader)
			if err != nil {
				return fmt.Errorf("unable to read initial stream window size on open: %w", err)
			}

			// If there's no capacity for additional streams in the backlog,
			// then enqueue a close message to reject the stream.
			if len(m.pendingInboundStreamIdentifiers) == m.configuration.AcceptBacklog {
				select {
				case m.enqueueClose <- streamIdentifier:
					continue
				case <-m.closed:
					return ErrMultiplexerClosed
				}
			}

			// Create the local end of the stream.
			stream := newStream(m, streamIdentifier, m.configuration.StreamReceiveWindow)

			// Set the stream's initial write window.
			stream.sendWindow = windowSize
			if windowSize > 0 {
				stream.sendWindowReady <- struct{}{}
			}

			// Register the stream.
			m.streamLock.Lock()
			m.streams[streamIdentifier] = stream
			m.streamLock.Unlock()

			// Enqueue the stream for acceptance.
			m.pendingInboundStreamIdentifiers <- streamIdentifier
		} else if kind == messageKindStreamAccept {
			// Decode the remote's initial receive window size.
			windowSize, err := binary.ReadUvarint(reader)
			if err != nil {
				return fmt.Errorf("unable to read initial stream window size on accept: %w", err)
			}

			// If the stream wasn't found locally, then we just have to assume
			// that the open request was already cancelled and that a close
			// response was already sent to the remote. In theory, there could
			// be misbehavior here from the remote, but we have no way to track
			// or detect it. In this case, we discard the message.
			if stream == nil {
				continue
			}

			// Verify that the stream wasn't already accepted or rejected.
			if isClosed(stream.established) {
				return errors.New("remote accepted the same stream twice")
			} else if isClosed(stream.remoteClosed) {
				return errors.New("remote accepted stream after closing it")
			}

			// Set the stream's initial write window. We don't need to lock the
			// write window at this point since the stream hasn't been returned
			// to the caller of OpenStream yet.
			stream.sendWindow = windowSize
			if windowSize > 0 {
				stream.sendWindowReady <- struct{}{}
			}

			// Mark the stream as accepted.
			close(stream.established)
		} else if kind == messageKindStreamData {
			// Decode the data length.
			if _, err := io.ReadFull(reader, lengthBuffer[:]); err != nil {
				return fmt.Errorf("unable to read data length: %w", err)
			}
			length := int(binary.BigEndian.Uint16(lengthBuffer[:]))
			if length == 0 {
				return errors.New("zero-length data received")
			}

			// If the stream wasn't found locally, then we just have to assume
			// that it was already closed locally and deregistered. In theory,
			// there could be misbehavior here from the remote, but we have no
			// way to track or detect it. In this case, we discard the data.
			if stream == nil {
				if _, err := reader.Discard(length); err != nil {
					return fmt.Errorf("unable to discard data: %w", err)
				}
				continue
			}

			// Verify that the stream has been established and isn't closed for
			// writing or closed.
			if !isClosed(stream.established) {
				return errors.New("data received for partially established stream")
			} else if isClosed(stream.remoteClosedWrite) {
				return errors.New("data received for write-closed stream")
			} else if isClosed(stream.remoteClosed) {
				return errors.New("data received for closed stream")
			}

			// Record the data.
			stream.receiveBufferLock.Lock()
			if _, err := stream.receiveBuffer.ReadNFrom(reader, length); err != nil {
				stream.receiveBufferLock.Unlock()
				if err == ring.ErrBufferFull {
					return errors.New("remote violated stream receive window")
				}
				return fmt.Errorf("unable to read stream data into buffer: %w", err)
			}
			if stream.receiveBuffer.Used() == length {
				stream.receiveBufferReady <- struct{}{}
			}
			stream.receiveBufferLock.Unlock()
		} else if kind == messageKindStreamWindowIncrement {
			// Decode the remote's receive window size increment.
			windowSizeIncrement, err := binary.ReadUvarint(reader)
			if err != nil {
				return fmt.Errorf("unable to read stream window size increment: %w", err)
			} else if windowSizeIncrement == 0 {
				return errors.New("zero-valued window increment received")
			}

			// If the stream wasn't found locally, then we just have to assume
			// that it was already closed locally and deregistered. In theory,
			// there could be misbehavior here from the remote, but we have no
			// way to track or detect it. In this case, we discard the message.
			if stream == nil {
				continue
			}

			// If this is an outbound stream, then ensure that the stream is
			// established (i.e. it's been accepted by the remote) before
			// allowing window increments. For inbound streams, we allow
			// adjustments to the window size before we accept the stream
			// locally, even though we don't utilize this feature at the moment.
			if streamIdentifierIsOutbound && !isClosed(stream.established) {
				return errors.New("window increment received for partially established outbound stream")
			}

			// Verify that the stream isn't already closed.
			if isClosed(stream.remoteClosed) {
				return errors.New("window increment received for closed stream")
			}

			// Increment the window.
			stream.sendWindowLock.Lock()
			if stream.sendWindow == 0 {
				stream.sendWindow = windowSizeIncrement
				stream.sendWindowReady <- struct{}{}
			} else {
				if math.MaxUint64-stream.sendWindow < windowSizeIncrement {
					stream.sendWindowLock.Unlock()
					return errors.New("window increment overflows maximum value")
				}
				stream.sendWindow += windowSizeIncrement
			}
			stream.sendWindowLock.Unlock()
		} else if kind == messageKindStreamCloseWrite {
			// If the stream wasn't found locally, then we just have to assume
			// that it was already closed locally and deregistered. In theory,
			// there could be misbehavior here from the remote, but we have no
			// way to track or detect it. In this case, we discard the message.
			if stream == nil {
				continue
			}

			// If this is an outbound stream, then ensure that the stream is
			// established (i.e. it's been accepted by the remote) before
			// allowing write closure. For inbound streams, we allow write
			// closure before we accept the stream locally, even though we don't
			// utilize this feature at the moment.
			if streamIdentifierIsOutbound && !isClosed(stream.established) {
				return errors.New("close write received for partially established outbound stream")
			}

			// Verify that the stream isn't already closed or closed for writes.
			if isClosed(stream.remoteClosed) {
				return errors.New("close write received for closed stream")
			} else if isClosed(stream.remoteClosedWrite) {
				return errors.New("close write received for the same stream twice")
			}

			// Signal write closure.
			close(stream.remoteClosedWrite)
		} else if kind == messageKindStreamClose {
			// If the stream wasn't found locally, then we just have to assume
			// that it was already closed locally and deregistered. In theory,
			// there could be misbehavior here from the remote, but we have no
			// way to track or detect it. In this case, we discard the message.
			if stream == nil {
				continue
			}

			// Verify that the stream isn't already closed.
			if isClosed(stream.remoteClosed) {
				return errors.New("close received the same stream twice")
			}

			// Signal closure.
			close(stream.remoteClosed)
		} else {
			panic("unhandled message kind")
		}
	}
}

// write is the entry point for the writer Goroutine.
func (m *Multiplexer) write(writer Carrier) error {
	// If outbound heartbeats are enabled, then create a ticker to regulate
	// heartbeat transmission, defer its shutdown, and craft a reusable
	// heartbeat message.
	var heartbeatTicker *time.Ticker
	var writeHeartbeat <-chan time.Time
	var heartbeat []byte
	if m.configuration.HeartbeatTransmitInterval > 0 {
		heartbeatTicker = time.NewTicker(m.configuration.HeartbeatTransmitInterval)
		defer heartbeatTicker.Stop()
		writeHeartbeat = heartbeatTicker.C
		heartbeat = []byte{byte(messageKindMultiplexerHeartbeat)}
	}

	// Loop until failure or multiplexer closure.
	for {
		select {
		case <-writeHeartbeat:
			if _, err := writer.Write(heartbeat); err != nil {
				return fmt.Errorf("unable to write heartbeat: %w", err)
			}
		case writeBuffer := <-m.writeBufferPending:
			if _, err := writeBuffer.WriteTo(writer); err != nil {
				return fmt.Errorf("unable to write message buffer: %w", err)
			}
			m.writeBufferAvailable <- writeBuffer
		case <-m.closed:
			return ErrMultiplexerClosed
		}
	}
}

// enqueue is the entry point for the state accumulation/transmission Goroutine.
func (m *Multiplexer) enqueue() {
	// Track pending updates.
	windowIncrements := make(map[uint64]uint64)
	writeCloses := make(map[uint64]bool)
	closes := make(map[uint64]bool)

	// Loop and process updates until failure.
	for {
		// Determine whether or not to poll for write buffer availability (based
		// on whether or not we have any pending updates).
		writeBufferAvailable := m.writeBufferAvailable
		if len(windowIncrements) == 0 && len(writeCloses) == 0 && len(closes) == 0 {
			writeBufferAvailable = nil
		}

		// Poll for a write buffer (if applicable), an update, or termination.
		// If we get a write buffer, then write as many updates as we can.
		select {
		case writeBuffer := <-writeBufferAvailable:
			for stream, amount := range windowIncrements {
				if writeBuffer.canEncodeStreamWindowIncrement() {
					writeBuffer.encodeStreamWindowIncrement(stream, amount)
					delete(windowIncrements, stream)
				} else {
					break
				}
			}
			for stream := range writeCloses {
				if writeBuffer.canEncodeStreamCloseWrite() {
					writeBuffer.encodeStreamCloseWrite(stream)
					delete(writeCloses, stream)
				} else {
					break
				}
			}
			for stream := range closes {
				if writeBuffer.canEncodeStreamClose() {
					writeBuffer.encodeStreamClose(stream)
					delete(closes, stream)
				} else {
					break
				}
			}
			m.writeBufferPending <- writeBuffer
		case increment := <-m.enqueueWindowIncrement:
			windowIncrements[increment.stream] = windowIncrements[increment.stream] + increment.amount
		case stream := <-m.enqueueCloseWrite:
			writeCloses[stream] = true
		case stream := <-m.enqueueClose:
			delete(windowIncrements, stream)
			delete(writeCloses, stream)
			closes[stream] = true
		case <-m.closed:
			return
		}
	}
}

// Addr implements net.Listener.Addr.
func (m *Multiplexer) Addr() net.Addr {
	return &multiplexerAddress{even: m.even}
}

// OpenStream opens a new stream, cancelling the open operation if the provided
// context is cancelled, an error occurs, or the multiplexer is closed. The
// context must not be nil. The context only regulates the lifetime of the open
// operation, not the stream itself.
func (m *Multiplexer) OpenStream(ctx context.Context) (*Stream, error) {
	// Create and register the local side of the stream. If we've already
	// exhausted local stream identifiers, then we can't open a new stream.
	m.streamLock.Lock()
	if m.nextOutboundStreamIdentifier == 0 {
		m.streamLock.Unlock()
		return nil, errors.New("local stream identifiers exhausted")
	}
	stream := newStream(m, m.nextOutboundStreamIdentifier, m.configuration.StreamReceiveWindow)
	m.streams[m.nextOutboundStreamIdentifier] = stream
	if math.MaxUint64-m.nextOutboundStreamIdentifier < 2 {
		m.nextOutboundStreamIdentifier = 0
	} else {
		m.nextOutboundStreamIdentifier += 2
	}
	m.streamLock.Unlock()

	// If we fail to establish the stream, then defer its closure. We can't use
	// the stream's established channel to check this because it could be closed
	// by the reader Goroutine after some other error aborts the opening.
	var sentOpenMessage, established bool
	defer func() {
		if !established {
			stream.close(sentOpenMessage)
		}
	}()

	// Write the open message and queue it for transmission.
	select {
	case writeBuffer := <-m.writeBufferAvailable:
		writeBuffer.encodeOpenMessage(stream.identifier, uint64(m.configuration.StreamReceiveWindow))
		m.writeBufferPending <- writeBuffer
		sentOpenMessage = true
	case <-ctx.Done():
		return nil, context.Canceled
	case <-m.closed:
		return nil, ErrMultiplexerClosed
	}

	// Wait for stream acceptance or rejection.
	select {
	case <-stream.established:
		established = true
		return stream, nil
	case <-stream.remoteClosed:
		return nil, ErrStreamRejected
	case <-ctx.Done():
		return nil, context.Canceled
	case <-m.closed:
		return nil, ErrMultiplexerClosed
	}
}

// errStaleInboundStream indicates that a stale inbound stream was encountered.
var errStaleInboundStream = errors.New("stale inbound stream")

// acceptOneStream is the internal stream accept method. It will only attempt
// one accept, and will return errStaleInboundStream if the accept request fails
// due to a stale inbound stream.
func (m *Multiplexer) acceptOneStream(ctx context.Context) (*Stream, error) {
	// Grab the oldest pending stream identifier.
	var streamIdentifier uint64
	select {
	case streamIdentifier = <-m.pendingInboundStreamIdentifiers:
	case <-ctx.Done():
		return nil, context.Canceled
	case <-m.closed:
		return nil, ErrMultiplexerClosed
	}

	// Grab the associated stream object, which is guaranteed to be non-nil.
	m.streamLock.Lock()
	stream := m.streams[streamIdentifier]
	m.streamLock.Unlock()

	// If we fail to establish the stream, then defer its closure. In this case
	// (unlike the opening case) we can use the stream's established channel to
	// check this because we're responsible for closing it.
	defer func() {
		if !isClosed(stream.established) {
			stream.Close()
		}
	}()

	// Wait for a write buffer to become available.
	var writeBuffer *messageBuffer
	select {
	case writeBuffer = <-m.writeBufferAvailable:
	case <-stream.remoteClosed:
		return nil, errStaleInboundStream
	case <-ctx.Done():
		return nil, context.Canceled
	case <-m.closed:
		return nil, ErrMultiplexerClosed
	}

	// Mark the stream as established. We need to do this before transmitting
	// the accept message because the other side might start sending messages
	// immediately and the reader Goroutine will want to confirm establishment
	// when processing those messages.
	close(stream.established)

	// Write the accept message and queue it for transmission.
	writeBuffer.encodeAcceptMessage(streamIdentifier, uint64(m.configuration.StreamReceiveWindow))
	m.writeBufferPending <- writeBuffer

	// Success.
	return stream, nil
}

// AcceptContext accepts an incoming stream.
func (m *Multiplexer) AcceptStream(ctx context.Context) (*Stream, error) {
	// Loop until we find a pending stream that's not stale or encounter some
	// other error.
	for {
		stream, err := m.acceptOneStream(ctx)
		if err == errStaleInboundStream {
			continue
		}
		return stream, err
	}
}

// Accept implements net.Listener.Accept. It is implemented as a wrapper around
// AcceptStream and simply casts the resulting stream to a net.Conn.
func (m *Multiplexer) Accept() (net.Conn, error) {
	stream, err := m.AcceptStream(context.Background())
	return stream, err
}

// Closed returns a channel that is closed when the multiplexer is closed (due
// to either internal failure or a manual call to Close).
func (m *Multiplexer) Closed() <-chan struct{} {
	return m.closed
}

// InternalError returns any internal error that caused the multiplexer to
// close (as indicated by closure of the result of Closed). It returns nil if
// Close was manually invoked.
func (m *Multiplexer) InternalError() error {
	m.internalErrorLock.RLock()
	defer m.internalErrorLock.RUnlock()
	return m.internalError
}

// closeWithError is the internal close method that allows for optional error
// reporting when closing.
func (m *Multiplexer) closeWithError(internalError error) (err error) {
	m.closeOnce.Do(func() {
		err = m.closer.Close()
		if internalError != nil {
			m.internalErrorLock.Lock()
			m.internalError = internalError
			m.internalErrorLock.Unlock()
		}
		close(m.closed)
	})
	return
}

// Close implements net.Listener.Close. Only the first call to Close will have
// any effect. Subsequent calls will behave as no-ops and return nil errors.
func (m *Multiplexer) Close() error {
	return m.closeWithError(nil)
}
