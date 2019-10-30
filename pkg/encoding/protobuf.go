package encoding

import (
	"bufio"
	"encoding/binary"
	"io"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/proto"
)

const (
	// protobufEncoderInitialBufferSize is the initial buffer size for encoders.
	protobufEncoderInitialBufferSize = 32 * 1024

	// protobufEncoderMaximumPersistentBufferSize is the maximum buffer size
	// that the encoder will keep allocated.
	protobufEncoderMaximumPersistentBufferSize = 1024 * 1024

	// protobufDecoderReaderBufferSize is the size to use for the buffered
	// reader in ProtobufDecoder.
	protobufDecoderReaderBufferSize = 32 * 1024

	// protobufDecoderInitialBufferSize is the initial buffer size for decoders.
	protobufDecoderInitialBufferSize = 32 * 1024

	// protobufDecoderMaximumAllowedMessageSize is the maximum message size that
	// we'll attempt to read from the wire.
	protobufDecoderMaximumAllowedMessageSize = 100 * 1024 * 1024

	// protobufDecoderMaximumPersistentBufferSize is the maximum buffer size
	// that the decoder will keep allocated.
	protobufDecoderMaximumPersistentBufferSize = 1024 * 1024
)

// LoadAndUnmarshalProtobuf loads data from the specified path and decodes it
// into the specified Protocol Buffers message.
func LoadAndUnmarshalProtobuf(path string, message proto.Message) error {
	return LoadAndUnmarshal(path, func(data []byte) error {
		return proto.Unmarshal(data, message)
	})
}

// MarshalAndSaveProtobuf marshals the specified Protocol Buffers message and
// saves it to the specified path.
func MarshalAndSaveProtobuf(path string, message proto.Message) error {
	return MarshalAndSave(path, func() ([]byte, error) {
		return proto.Marshal(message)
	})
}

// ProtobufEncoder is a stream encoder for Protocol Buffers messages.
type ProtobufEncoder struct {
	// writer is the underlying writer.
	writer io.Writer
	// buffer is a reusable Protocol Buffers encoding buffer.
	buffer *proto.Buffer
}

// NewProtobufEncoder creates a new Protocol Buffers stream encoder.
func NewProtobufEncoder(writer io.Writer) *ProtobufEncoder {
	return &ProtobufEncoder{
		writer: writer,
		buffer: proto.NewBuffer(make([]byte, 0, protobufEncoderInitialBufferSize)),
	}
}

// EncodeWithoutFlush encodes a length-prefixed Protocol Buffers message into
// the encoder's internal buffer, but does not write this data to the underlying
// stream.
func (e *ProtobufEncoder) EncodeWithoutFlush(message proto.Message) error {
	// Encode the message with a length prefix. The 64-bit unsigned varint
	// encoding used by this method is the same used by the encoding/binary
	// package.
	// TODO: This method is internally a bit inefficient because it computes the
	// encoded message size twice. Unfortunately there's no way to get around
	// this without resorting to using private interfaces in the generated
	// Protocol Buffers code. We should file an issue about this though.
	if err := e.buffer.EncodeMessage(message); err != nil {
		return errors.Wrap(err, "unable to encode message")
	}

	// Success.
	return nil
}

// Flush writes the contents of the encoder's internal buffer, if any, to the
// underlying stream.
func (e *ProtobufEncoder) Flush() error {
	// Extract the data waiting in the buffer.
	data := e.buffer.Bytes()

	// Write the data to the wire if there is any.
	if len(data) > 0 {
		if _, err := e.writer.Write(data); err != nil {
			return errors.Wrap(err, "unable to write message")
		}
	}

	// Check if the buffer's capacity has grown beyond what we're willing to
	// carry around. If so, reset the buffer to the maximum persistent buffer
	// size. Otherwise, reset the buffer so that it continues to use the same
	// slice.
	if cap(data) > protobufEncoderMaximumPersistentBufferSize {
		e.buffer.SetBuf(make([]byte, 0, protobufEncoderMaximumPersistentBufferSize))
	} else {
		e.buffer.Reset()
	}

	// Success.
	return nil
}

// Encode encodes a length-prefixed Protocol Buffers message into the encoder's
// internal buffer and writes this data to the underlying stream.
func (e *ProtobufEncoder) Encode(message proto.Message) error {
	// Encode the message.
	if err := e.EncodeWithoutFlush(message); err != nil {
		return err
	}

	// Write the message to the wire.
	return e.Flush()
}

// ProtobufDecoder is a stream decoder for Protocol Buffers messages. For
// performance reasons, this type wraps the underlying stream in a buffering
// reader, which means that the decoder should persist for the lifetime of the
// stream (since there's no way to know how much data has been read from the
// stream). For single-message decoding that won't over-read a stream, use
// DecodeProtobuf.
type ProtobufDecoder struct {
	// reader is a buffered reader wrapping the underlying reader.
	reader *bufio.Reader
	// buffer is a reusable receive buffer for decoding messages.
	buffer []byte
}

// NewProtobufDecoder creates a new Protocol Buffers stream decoder.
func NewProtobufDecoder(reader io.Reader) *ProtobufDecoder {
	return &ProtobufDecoder{
		reader: bufio.NewReaderSize(reader, protobufDecoderReaderBufferSize),
		buffer: make([]byte, protobufDecoderInitialBufferSize),
	}
}

// bufferWithSize returns a buffer with the specified size, opting to reuse a
// cached buffer if possible.
func (d *ProtobufDecoder) bufferWithSize(size int) []byte {
	// If we can satisfy this request with our existing buffer, then use that.
	if cap(d.buffer) >= size {
		return d.buffer[:size]
	}

	// Otherwise allocate a new buffer.
	result := make([]byte, size)

	// If this buffer doesn't exceed the maximum size that we're willing to keep
	// around in memory, then store it.
	if size <= protobufDecoderMaximumPersistentBufferSize {
		d.buffer = result
	}

	// Done.
	return result
}

// Decode decodes a length-prefixed Protocol Buffers message from the underlying
// stream.
func (d *ProtobufDecoder) Decode(message proto.Message) error {
	// Read the next message length.
	length, err := binary.ReadUvarint(d.reader)
	if err != nil {
		return errors.Wrap(err, "unable to read message length")
	}

	// Check if the message is too long to read.
	if length > protobufDecoderMaximumAllowedMessageSize {
		return errors.New("message size too large")
	}

	// Grab a buffer to read the message.
	messageBytes := d.bufferWithSize(int(length))

	// Read the message bytes.
	if _, err := io.ReadFull(d.reader, messageBytes); err != nil {
		return errors.Wrap(err, "unable to read message")
	}

	// Unmarshal the message.
	if err := proto.Unmarshal(messageBytes, message); err != nil {
		return errors.Wrap(err, "unable to unmarshal message")
	}

	// Success.
	return nil
}

// EncodeProtobuf writes a single Protocol Buffers messages that can be read by
// ProtobufDecoder or ReadProtobuf. It is a useful shorthand for creating a
// ProtobufEncoder and writing a single message. For multiple message sends, it
// is far more efficient to use a ProtobufEncoder directly.
func EncodeProtobuf(writer io.Writer, message proto.Message) error {
	return NewProtobufEncoder(writer).Encode(message)
}

// simpleByteReader is a naïve io.ByteReader implementation that operates on top
// of an io.Reader. It is not efficient, so it should not be used except for
// single message reads where a buffered reader would not be appropriate.
type simpleByteReader struct {
	// reader is the underlying reader.
	reader io.Reader
}

// ReadByte implements io.ByteReader.ReadByte.
func (r *simpleByteReader) ReadByte() (byte, error) {
	var data [1]byte
	if _, err := io.ReadFull(r.reader, data[:]); err != nil {
		return 0, err
	}
	return data[0], nil
}

// DecodeProtobuf reads and decodes a single Protocol Buffers message as
// transmitted by ProtobufEncoder or WriteProtobuf. It is useful in cases where
// a ProtobufDecoder cannot be used because the underlying stream will send data
// after the Protocol Buffers message that could be gobbled up by the decoder's
// buffered reads, but it is not an efficient implementation, and thus should
// only be used for a fixed number of messages (e.g. a single initialization
// message).
func DecodeProtobuf(reader io.Reader, message proto.Message) error {
	// Read the next message length.
	length, err := binary.ReadUvarint(&simpleByteReader{reader})
	if err != nil {
		return errors.Wrap(err, "unable to read message length")
	}

	// Check if the message is too long to read.
	if length > protobufDecoderMaximumAllowedMessageSize {
		return errors.New("message size too large")
	}

	// Create a buffer to read the message.
	messageBytes := make([]byte, length)

	// Read the message bytes.
	if _, err := io.ReadFull(reader, messageBytes); err != nil {
		return errors.Wrap(err, "unable to read message")
	}

	// Unmarshal the message.
	if err := proto.Unmarshal(messageBytes, message); err != nil {
		return errors.Wrap(err, "unable to unmarshal message")
	}

	// Success.
	return nil
}
