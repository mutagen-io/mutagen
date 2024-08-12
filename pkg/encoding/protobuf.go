package encoding

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	"github.com/mutagen-io/mutagen/pkg/stream"
)

const (
	// protobufEncoderInitialBufferSize is the initial buffer size for encoders.
	protobufEncoderInitialBufferSize = 32 * 1024

	// protobufEncoderMaximumPersistentBufferSize is the maximum buffer size
	// that the encoder will keep allocated.
	protobufEncoderMaximumPersistentBufferSize = 1024 * 1024

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
func MarshalAndSaveProtobuf(path string, message proto.Message, logger *logging.Logger) error {
	return MarshalAndSave(path, logger, func() ([]byte, error) {
		return proto.Marshal(message)
	})
}

// ProtobufEncoder is a stream encoder for Protocol Buffers messages.
type ProtobufEncoder struct {
	// writer is the underlying writer.
	writer io.Writer
	// buffer is a reusable encoding buffer.
	buffer []byte
	// sizer is a Protocol Buffers marshaling configuration for computing sizes.
	sizer proto.MarshalOptions
	// encoder is a Protocol Buffers marshaling configuration for encoding.
	encoder proto.MarshalOptions
}

// NewProtobufEncoder creates a new Protocol Buffers stream encoder.
func NewProtobufEncoder(writer io.Writer) *ProtobufEncoder {
	return &ProtobufEncoder{
		writer:  writer,
		buffer:  make([]byte, 0, protobufEncoderInitialBufferSize),
		sizer:   proto.MarshalOptions{},
		encoder: proto.MarshalOptions{UseCachedSize: true},
	}
}

// Encode encodes and writes a length-prefixed Protocol Buffers message to the
// underlying stream. If this fails, the encoder should be considered corrupted.
func (e *ProtobufEncoder) Encode(message proto.Message) error {
	// Always make sure that the buffer's capacity stays within the limit of
	// what we're willing to carry around once we're done.
	defer func() {
		if cap(e.buffer) > protobufEncoderMaximumPersistentBufferSize {
			e.buffer = make([]byte, 0, protobufEncoderMaximumPersistentBufferSize)
		} else {
			e.buffer = e.buffer[:0]
		}
	}()

	// Encode the message size.
	e.buffer = protowire.AppendVarint(e.buffer, uint64(e.sizer.Size(message)))

	// Encode the message.
	var err error
	e.buffer, err = e.encoder.MarshalAppend(e.buffer, message)
	if err != nil {
		return fmt.Errorf("unable to marshal message: %w", err)
	}

	// Write the data to the wire.
	if _, err = e.writer.Write(e.buffer); err != nil {
		return fmt.Errorf("unable to write message: %w", err)
	}

	// Success.
	return nil
}

// ProtobufDecoder is a stream decoder for Protocol Buffers messages.
type ProtobufDecoder struct {
	// reader is the underlying reader.
	reader stream.DualModeReader
	// buffer is a reusable receive buffer for decoding messages.
	buffer []byte
}

// NewProtobufDecoder creates a new Protocol Buffers stream decoder.
func NewProtobufDecoder(reader stream.DualModeReader) *ProtobufDecoder {
	return &ProtobufDecoder{
		reader: reader,
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
// stream. If this fails, the decoder should be considered corrupted.
func (d *ProtobufDecoder) Decode(message proto.Message) error {
	// Read the next message length.
	length, err := binary.ReadUvarint(d.reader)
	if err != nil {
		return fmt.Errorf("unable to read message length: %w", err)
	}

	// Check if the message is too long to read.
	if length > protobufDecoderMaximumAllowedMessageSize {
		return errors.New("message size too large")
	}

	// Grab a buffer to read the message.
	messageBytes := d.bufferWithSize(int(length))

	// Read the message bytes.
	if _, err := io.ReadFull(d.reader, messageBytes); err != nil {
		return fmt.Errorf("unable to read message: %w", err)
	}

	// Unmarshal the message.
	if err := proto.Unmarshal(messageBytes, message); err != nil {
		return fmt.Errorf("unable to unmarshal message: %w", err)
	}

	// Success.
	return nil
}

// EncodeProtobuf encodes a single Protocol Buffers message that can be read by
// ProtobufDecoder or DecodeProtobuf. It is a useful shorthand for creating a
// ProtobufEncoder and writing a single message. For multiple message sends, it
// is far more efficient to use a ProtobufEncoder directly and repeatedly.
func EncodeProtobuf(writer io.Writer, message proto.Message) error {
	return NewProtobufEncoder(writer).Encode(message)
}

// DecodeProtobuf reads and decodes a single Protocol Buffers message as written
// by ProtobufEncoder or EncodeProtobuf. It is a useful shorthand for creating a
// ProtobufDecoder and reading a single message. For multiple message reads, it
// is far more efficient to use a ProtobufDecoder directly and repeatedly.
func DecodeProtobuf(reader stream.DualModeReader, message proto.Message) error {
	return NewProtobufDecoder(reader).Decode(message)
}
