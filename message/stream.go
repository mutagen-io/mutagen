package message

import (
	"encoding/gob"
	"io"

	"github.com/golang/snappy"
)

type messageStream struct {
	decoder *gob.Decoder
	encoder *gob.Encoder
	flusher *snappy.Writer
}

func (s *messageStream) Decode(message interface{}) error {
	return s.decoder.Decode(message)
}

func (s *messageStream) Encode(message interface{}) error {
	// Encode the message.
	if err := s.encoder.Encode(message); err != nil {
		return err
	}

	// Flush if necessary.
	if s.flusher != nil {
		if err := s.flusher.Flush(); err != nil {
			return err
		}
	}

	// Success.
	return nil
}

// MessageStream provides message transmission and reception using the semantics
// of the encoding/gob package.
type MessageStream interface {
	Decode(interface{}) error
	Encode(interface{}) error
}

// NewMessageStream constructs a message stream using a raw byte stream.
func NewMessageStream(raw io.ReadWriter) MessageStream {
	return &messageStream{
		decoder: gob.NewDecoder(raw),
		encoder: gob.NewEncoder(raw),
	}
}

// NewCompressedMessageStream constructs a compressed message stream using a raw
// byte stream.
func NewCompressedMessageStream(raw io.ReadWriter) MessageStream {
	// Create a decompressing decoder.
	decoder := gob.NewDecoder(snappy.NewReader(raw))

	// Create a compressing writer.
	writer := snappy.NewBufferedWriter(raw)
	encoder := gob.NewEncoder(writer)

	// Create the message stream.
	return &messageStream{decoder, encoder, writer}
}
