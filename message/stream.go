package message

import (
	"encoding/gob"
	"io"

	"github.com/golang/snappy"
)

type Stream struct {
	decoder    *gob.Decoder
	encoder    *gob.Encoder
	compressor *snappy.Writer
}

// NewStream constructs a message stream on top of a raw byte stream with
// optional compression.
func NewStream(raw io.ReadWriter, compress bool) *Stream {
	// Extract the reader, wrapping it in a decompressor if necessary.
	var reader io.Reader = raw
	if compress {
		reader = snappy.NewReader(raw)
	}

	// Extract the writer, wrapping it in a compressor if necessary.
	var compressor *snappy.Writer
	var writer io.Writer = raw
	if compress {
		compressor = snappy.NewBufferedWriter(raw)
		writer = compressor
	}

	// Create the message stream.
	return &Stream{
		decoder:    gob.NewDecoder(reader),
		encoder:    gob.NewEncoder(writer),
		compressor: compressor,
	}
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
	if s.compressor != nil {
		if err := s.compressor.Flush(); err != nil {
			return err
		}
	}

	// Success.
	return nil
}
