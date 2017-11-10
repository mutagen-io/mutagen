package rpc

import (
	"encoding/gob"
	"fmt"
	"io"
	"net"

	"github.com/pkg/errors"
)

type ClientStream interface {
	Send(interface{}) error
	// TODO: Document that Receive returns io.EOF unmodified (so long as the
	// stream is closed on a message boundary) and add a test to ensure this.
	// Prompting (and potentially other code) relies on this to identify clean
	// termination.
	Receive(interface{}) error
	// Close closes the stream. It may be called concurrently with the Send and
	// Receive methods, which it will unblock.
	Close() error
}

type HandlerStream interface {
	Send(interface{}) error
	// TODO: Document that Receive returns io.EOF unmodified (so long as the
	// stream is closed on a message boundary) and add a test to ensure this.
	// There isn't any code currently relying on this behavior, but we should
	// keep it symmetric with ClientStream because it's useful for the same
	// reason.
	Receive(interface{}) error
}

type RemoteError struct {
	message string
}

func (e *RemoteError) Error() string {
	return fmt.Sprintf("remote error: %s", e.message)
}

type messageHeader struct {
	Errored bool
	Error   string
}

type stream struct {
	stream  net.Conn
	errored bool
	encoder *gob.Encoder
	decoder *gob.Decoder
}

func newStream(connection net.Conn) *stream {
	return &stream{
		stream:  connection,
		encoder: gob.NewEncoder(connection),
		decoder: gob.NewDecoder(connection),
	}
}

func (s *stream) Send(value interface{}) error {
	// Verify that the stream isn't errored.
	if s.errored {
		return errors.New("stream is errored")
	}

	// Encode the header.
	if err := s.encoder.Encode(messageHeader{}); err != nil {
		s.errored = true
		return errors.Wrap(err, "unable to encode message header")
	}

	// Encode the message.
	if err := s.encoder.Encode(value); err != nil {
		s.errored = true
		return errors.Wrap(err, "unable to encode message")
	}

	// Success.
	return nil
}

func (s *stream) markError(err error) error {
	// Verify that the stream isn't errored.
	if s.errored {
		return errors.New("stream is errored")
	}

	// Mark the stream as errored.
	s.errored = true

	// Create the header.
	header := messageHeader{Errored: true, Error: err.Error()}

	// Transmit the header.
	if err := s.encoder.Encode(header); err != nil {
		s.stream.Close()
		return errors.Wrap(err, "unable to encode message header")
	}

	// Success.
	return nil
}

func (s *stream) Receive(value interface{}) error {
	// Verify that the stream isn't errored.
	if s.errored {
		return errors.New("stream is errored")
	}

	// Decode the header. We pass certain sentinel errors through unwrapped here
	// because they are useful for providing semantic meaning.
	var header messageHeader
	if err := s.decoder.Decode(&header); err != nil {
		s.errored = true
		if err == io.EOF {
			return err
		}
		return errors.Wrap(err, "unable to decode message header")
	}

	// Check the header for errors.
	if header.Errored {
		s.errored = true
		return &RemoteError{header.Error}
	}

	// Decode the message. Here we don't pass any sentinel errors through
	// because we treat headers and messages as atomic units - they should come
	// together. If one arrives and the other returns with an EOF, then the
	// connection has been broken unnaturally.
	if err := s.decoder.Decode(value); err != nil {
		s.errored = true
		return errors.Wrap(err, "unable to decode message")
	}

	// Success.
	return nil
}

func (s *stream) Close() error {
	return s.stream.Close()
}
