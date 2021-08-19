package multiplexing

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/mutagen-io/mutagen/pkg/multiplexing/ring"
)

// messageKind encodes a message kind on the wire.
type messageKind byte

const (
	// messageKindMultiplexerHeartbeat indicates a multiplexer heartbeat
	// message. The message is structured as follows:
	// - Message kind (byte)
	messageKindMultiplexerHeartbeat messageKind = iota
	// messageKindStreamOpen indicates a stream open message. The message is
	// structured as follows:
	// - Message kind (byte)
	// - Stream identifier (uvarint64)
	// - Initial remote stream receive window size (uvarint64)
	messageKindStreamOpen
	// messageKindStreamOpen indicates a stream accept message. The message is
	// structured as follows:
	// - Message kind (byte)
	// - Stream identifier (uvarint64)
	// - Initial remote stream receive window size (uvarint64)
	messageKindStreamAccept
	// messageKindStreamData indicates a stream data message. The message is
	// structured as follows:
	// - Message kind (byte)
	// - Stream identifier (uvarint64)
	// - Data length (uint16 (network byte order))
	// - Data (bytes)
	messageKindStreamData
	// messageKindStreamWindowIncrement indicates a stream receive window size
	// increment message. The message is structured as follows:
	// - Message kind (byte)
	// - Stream identifier (uvarint64)
	// - Increment amount (uvarint64)
	messageKindStreamWindowIncrement
	// messageKindStreamCloseWrite indicates a stream write close message. The
	// message is structured as follows:
	// - Message kind (byte)
	// - Stream identifier (uvarint64)
	messageKindStreamCloseWrite
	// messageKindStreamClose indicates a stream close message. The message is
	// structured as follows:
	// - Message kind (byte)
	// - Stream identifier (uvarint64)
	messageKindStreamClose
)

const (
	// messageKindStreamOpenMaximumSize is the maximum size of a stream open
	// message.
	messageKindStreamOpenMaximumSize = 1 + binary.MaxVarintLen64 + binary.MaxVarintLen64
	// messageKindStreamAcceptMaximumSize is the maximum size of a stream accept
	// message.
	messageKindStreamAcceptMaximumSize = 1 + binary.MaxVarintLen64 + binary.MaxVarintLen64
	// messageKindStreamDataMaximumSize is the maximum size of a stream data
	// message.
	messageKindStreamDataMaximumSize = 1 + binary.MaxVarintLen64 + 2 + math.MaxUint16
	// messageKindStreamWindowIncrementMaximumSize is the maximum size of a
	// stream window increment message.
	messageKindStreamWindowIncrementMaximumSize = 1 + binary.MaxVarintLen64 + binary.MaxVarintLen64
	// messageKindStreamCloseWriteMaximumSize is the maximum size of a stream
	// close write message.
	messageKindStreamCloseWriteMaximumSize = 1 + binary.MaxVarintLen64
	// messageKindStreamCloseMaximumSize is the maximum size of a stream close
	// message.
	messageKindStreamCloseMaximumSize = 1 + binary.MaxVarintLen64

	// maximumMessageSize is the maximum size of any single messsage.
	maximumMessageSize = messageKindStreamDataMaximumSize

	// maximumStreamDataBlockSize is the maximum size (in bytes) for a single
	// block of stream data sent with a stream data message. It is determined
	// by the use of a 16-bit unsigned integer for encoding its length.
	maximumStreamDataBlockSize = math.MaxUint16
)

// messageBuffer is a reusable buffer type for encoding and transmitting
// protocol messages.
type messageBuffer struct {
	// buffer is the underlying buffer used for storage.
	buffer *ring.Buffer
	// varint64Buffer is a reusable buffer for encoding variable length integers
	// up to 64-bits. It is also used for encoding 16-bit unsigned integers to
	// network byte order.
	varint64Buffer []byte
}

// newMessageBuffer creates a new message buffer. It is guaranteed to have
// enough capacity to write any single message.
func newMessageBuffer() *messageBuffer {
	return &messageBuffer{
		buffer:         ring.NewBuffer(maximumMessageSize),
		varint64Buffer: make([]byte, binary.MaxVarintLen64),
	}
}

// ensureSufficientFreeSpace panics if the buffer doesn't contain at least the
// specified amount of free space.
func (b *messageBuffer) ensureSufficientFreeSpace(amount int) {
	if b.buffer.Free() < amount {
		panic("buffer not guaranteed to have sufficient free space")
	}
}

// writeUvarint is an internal utility function used to write unsigned variable
// length integers up to 64-bits.
func (b *messageBuffer) writeUvarint(value uint64) {
	length := binary.PutUvarint(b.varint64Buffer, value)
	b.buffer.Write(b.varint64Buffer[:length])
}

// writeUint16 is an internal utility function used to write unsigned 16-bit
// integers.
func (b *messageBuffer) writeUint16(value uint16) {
	binary.BigEndian.PutUint16(b.varint64Buffer[:2], value)
	b.buffer.Write(b.varint64Buffer[:2])
}

// WriteTo implements io.WriterTo.WriteTo.
func (b *messageBuffer) WriteTo(writer io.Writer) (int64, error) {
	return b.buffer.WriteTo(writer)
}

// encodeOpenMessage encodes a stream open message to the message buffer. It
// will panic if the buffer does not have sufficient free space.
func (b *messageBuffer) encodeOpenMessage(stream, window uint64) {
	b.ensureSufficientFreeSpace(messageKindStreamOpenMaximumSize)
	b.buffer.WriteByte(byte(messageKindStreamOpen))
	b.writeUvarint(stream)
	b.writeUvarint(window)
}

// encodeAcceptMessage encodes a stream accept message to the message buffer. It
// will panic if the buffer does not have sufficient free space.
func (b *messageBuffer) encodeAcceptMessage(stream, window uint64) {
	b.ensureSufficientFreeSpace(messageKindStreamAcceptMaximumSize)
	b.buffer.WriteByte(byte(messageKindStreamAccept))
	b.writeUvarint(stream)
	b.writeUvarint(window)
}

// encodeStreamDataMessage encodes a stream data message to the buffer. It will
// panic if the buffer does not have sufficient free space or if the data block
// is larger than maximumStreamDataBlockSize.
func (b *messageBuffer) encodeStreamDataMessage(stream uint64, data []byte) {
	b.ensureSufficientFreeSpace(messageKindStreamDataMaximumSize)
	if len(data) > maximumStreamDataBlockSize {
		panic("data block too large")
	}
	b.buffer.WriteByte(byte(messageKindStreamData))
	b.writeUvarint(stream)
	b.writeUint16(uint16(len(data)))
	b.buffer.Write(data)
}

// canEncodeStreamWindowIncrement returns whether or not a call to
// encodeStreamWindowIncrement is guaranteed to have sufficient free space.
func (b *messageBuffer) canEncodeStreamWindowIncrement() bool {
	return b.buffer.Free() >= messageKindStreamWindowIncrementMaximumSize
}

// encodeStreamWindowIncrement encodes a stream window increment message to the
// buffer. It will panic if the buffer does not have sufficient free space.
func (b *messageBuffer) encodeStreamWindowIncrement(stream, amount uint64) {
	b.ensureSufficientFreeSpace(messageKindStreamWindowIncrementMaximumSize)
	b.buffer.WriteByte(byte(messageKindStreamWindowIncrement))
	b.writeUvarint(stream)
	b.writeUvarint(amount)
}

// canEncodeStreamCloseWrite returns whether or not a call to
// encodeStreamCloseWrite is guaranteed to have sufficient free space.
func (b *messageBuffer) canEncodeStreamCloseWrite() bool {
	return b.buffer.Free() >= messageKindStreamCloseWriteMaximumSize
}

// encodeStreamCloseWrite encodes a stream half-closure message to the buffer.
// It will panic if the buffer does not have sufficient free space.
func (b *messageBuffer) encodeStreamCloseWrite(stream uint64) {
	b.ensureSufficientFreeSpace(messageKindStreamCloseWriteMaximumSize)
	b.buffer.WriteByte(byte(messageKindStreamCloseWrite))
	b.writeUvarint(stream)
}

// canEncodeStreamClose returns whether or not a call to encodeStreamClose is
// guaranteed to have sufficient free space.
func (b *messageBuffer) canEncodeStreamClose() bool {
	return b.buffer.Free() >= messageKindStreamCloseMaximumSize
}

// encodeStreamClose encodes a stream closure message to the buffer. It will
// panic if the buffer does not have sufficient free space.
func (b *messageBuffer) encodeStreamClose(stream uint64) {
	b.ensureSufficientFreeSpace(messageKindStreamCloseMaximumSize)
	b.buffer.WriteByte(byte(messageKindStreamClose))
	b.writeUvarint(stream)
}
