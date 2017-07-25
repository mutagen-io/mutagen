package message

import (
	"encoding/gob"
	"io"

	"github.com/golang/snappy"
)

type Stream struct {
	decoder *gob.Decoder
	encoder *gob.Encoder
	flusher *snappy.Writer
}

func (s *Stream) Decode(message interface{}) error {
	return s.decoder.Decode(message)
}

func (s *Stream) Encode(message interface{}) error {
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

// NewStream constructs a message stream using a raw byte stream.
func NewStream(raw io.ReadWriter) *Stream {
	return &Stream{
		decoder: gob.NewDecoder(raw),
		encoder: gob.NewEncoder(raw),
	}
}

// NewCompresseStream constructs a compressed message stream using a raw byte
// stream.
func NewCompressedStream(raw io.ReadWriter) *Stream {
	// Create a decompressing decoder.
	decoder := gob.NewDecoder(snappy.NewReader(raw))

	// Create a compressing writer.
	writer := snappy.NewBufferedWriter(raw)
	encoder := gob.NewEncoder(writer)

	// Create the message stream.
	return &Stream{decoder, encoder, writer}
}
