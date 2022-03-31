package remote

import (
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/stream"
	"github.com/mutagen-io/mutagen/pkg/synchronization/rsync"
)

// protobufRsyncEncoder implements rsync.Encoder using Protocol Buffers.
type protobufRsyncEncoder struct {
	// encoder is the underlying Protocol Buffers encoder.
	encoder *encoding.ProtobufEncoder
	// flusher flushes the underlying stream.
	flusher stream.Flusher
	// error stores any previously encountered transmission error.
	error error
}

// Encode implements rsync.Encoder.Encode.
func (e *protobufRsyncEncoder) Encode(transmission *rsync.Transmission) error {
	// Check for previous errors.
	if e.error != nil {
		return fmt.Errorf("previous error encountered: %w", e.error)
	}

	// Encode the transmission.
	e.error = e.encoder.Encode(transmission)
	return e.error
}

// Finalize implements rsync.Encoder.Finalize.
func (e *protobufRsyncEncoder) Finalize() error {
	// If an error has occurred, then there's nothing to do.
	if e.error != nil {
		return nil
	}

	// Otherwise, attempt to flush the compressor.
	if err := e.flusher.Flush(); err != nil {
		return fmt.Errorf("unable to flush encoded messages: %w", err)
	}

	// Success.
	return nil
}

// protobufRsyncDecoder implements rsync.Decoder using Protocol Buffers.
type protobufRsyncDecoder struct {
	// decoder is the underlying Protocol Buffers decoder.
	decoder *encoding.ProtobufDecoder
}

// Decode implements rsync.Decoder.Decode.
func (d *protobufRsyncDecoder) Decode(transmission *rsync.Transmission) error {
	// TODO: This is not particularly efficient because the Protocol Buffers
	// decoding implementation doesn't reuse existing capacity in operation data
	// buffers. This is something that needs to be fixed upstream, but we should
	// file an issue. Once it's done, nothing on our end needs to change except
	// to update the Protocol Buffers runtime.
	return d.decoder.Decode(transmission)
}

// Finalize implements rsync.Decoder.Finalize.
func (d *protobufRsyncDecoder) Finalize() error {
	return nil
}
