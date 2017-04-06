package framing

import (
	"bufio"
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
)

const (
	// maximumMessageSize is the maximum message size that the framing protocol
	// will transmit or receive. It is somewhat arbitrary, but chosen to avoid
	// exhausting memory.
	maximumMessageSize = 25 * 1024 * 1024
	// maximumMessageUvarintLength is the number of bytes that
	// maximumMessageSize takes to represent in unsigned varint encoding.
	maximumMessageUvarintLength = 4
	// reusableBufferSize is the size of the buffer that Encoder and Decoder
	// will allocate and retain to encode/decode messages. Messages larger than
	// this (but less than or equal to maximumMessageSize) will have a temporary
	// buffer allocated for their encoding/decoding, but this buffer won't
	// persist beyond the Encode/Decode call. Ideally, this size should be small
	// enough that it doesn't cost too much memory, especially when multiple
	// Encoders and Decoders exists, but large enough to accomodate most common
	// messages.
	reusableBufferSize = 100 * 1024
)

// Encodable is the interface that messages must satisfy to be framed. It is
// satisfied by all gogo/protobuf messages with generated sizing and marshalling
// methods.
type Encodable interface {
	// Size yields the size of the message when encoded.
	Size() int
	// MarshalTo encodes the message into the provided buffer, returning the
	// number of bytes written (which must be the same as what was returned from
	// the Size method) and/or an error.
	MarshalTo([]byte) (int, error)
}

// Encoder provides framed message encoding.
type Encoder struct {
	// writer is the underlying stream on which framed messages will be sent.
	writer io.Writer
	// buffer is a staging area for building framed messages. It has enough
	// space for the header and maximum message length.
	buffer []byte
}

// NewEncoder creates a new framing encoder.
func NewEncoder(writer io.Writer) *Encoder {
	return &Encoder{
		writer: writer,
		buffer: make([]byte, maximumMessageUvarintLength+reusableBufferSize),
	}
}

// Encode transmits a framed encoded message.
func (e *Encoder) Encode(message Encodable) error {
	// Compute the required size and ensure that it's frameable.
	size := message.Size()
	if size > maximumMessageSize {
		return errors.New("encoded message too large to frame")
	}

	// Check if we can use our internal buffer for encoding. If not, allocate a
	// temporary one. If s <= reusableBufferSize, then we know that both the
	// varint length and message size will fit into the internal buffer since
	// reusableBufferSize < maximumMessageSize and we've allocated out internal
	// buffer to support uvarint encodings of lengths up to maximumMessageSize.
	buffer := e.buffer
	if size > reusableBufferSize {
		buffer = make([]byte, maximumMessageUvarintLength+size)
	}

	// Encode the header.
	headerSize := binary.PutUvarint(buffer, uint64(size))

	// Encode the message.
	if count, err := message.MarshalTo(buffer[headerSize : headerSize+size]); err != nil {
		return errors.Wrap(err, "unable to serialize message")
	} else if count != size {
		return errors.New("encoded message had unexpected size")
	}

	// Transmit the framed message.
	if _, err := e.writer.Write(buffer[:headerSize+size]); err != nil {
		return errors.Wrap(err, "unable to transmit serialized message")
	}

	// Success.
	return nil
}

// Decodable is the interface that messages must satisfy to be deframed. It is
// satisfied by all gogo/protobuf messages with generated unmarshalling methods.
type Decodable interface {
	// Unmarshal decodes the message (into the callee) from the provided buffer,
	// returning any error that occurs.
	Unmarshal([]byte) error
}

// Decoder provides framed message decoding.
type Decoder struct {
	// reader is the underlying stream from which framed messages will be read.
	// We buffer it so that we can perform header decoding.
	reader *bufio.Reader
	// buffer is a staging area for receiving serialized messages.
	buffer []byte
}

// NewEncoder creates a new framing encoder.
func NewDecoder(reader io.Reader) *Decoder {
	return &Decoder{
		reader: bufio.NewReader(reader),
		buffer: make([]byte, reusableBufferSize),
	}
}

func (d *Decoder) DecodeTo(message Decodable) error {
	// Read the header.
	size, err := binary.ReadUvarint(d.reader)
	if err != nil {
		return errors.Wrap(err, "unable to read header")
	} else if size > maximumMessageSize {
		return errors.New("message too large to receive")
	}

	// Check if we can use our internal buffer for encoding. If not, allocate a
	// temporary one.
	buffer := d.buffer
	if size > reusableBufferSize {
		buffer = make([]byte, size)
	}

	// Read the message.
	if _, err := io.ReadFull(d.reader, buffer[:size]); err != nil {
		return errors.Wrap(err, "unable to read message body")
	}

	// Deserialize the message.
	if err := message.Unmarshal(buffer[:size]); err != nil {
		return errors.Wrap(err, "unable to deserialize message")
	}

	// Success.
	return nil
}
