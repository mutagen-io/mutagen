package message

import (
	"encoding/gob"
	"io"
)

type Stream struct {
	*gob.Decoder
	*gob.Encoder
}

// NewStream constructs a message stream on top of a raw byte stream.
func NewStream(raw io.ReadWriter) *Stream {
	return &Stream{gob.NewDecoder(raw), gob.NewEncoder(raw)}
}
