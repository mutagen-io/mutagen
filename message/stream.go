package message

import (
	"encoding/gob"
	"io"
)

// MessageStream provides message transmission and reception using the semantics
// of the encoding/gob package.
type MessageStream interface {
	Decode(interface{}) error
	Encode(interface{}) error
}

// messageStream is the internal representation of the MessageStream interface.
type messageStream struct {
	*gob.Decoder
	*gob.Encoder
}

// NewMessageStream constructs a message stream using a raw byte stream.
func NewMessageStream(raw io.ReadWriter) MessageStream {
	return &messageStream{gob.NewDecoder(raw), gob.NewEncoder(raw)}
}
