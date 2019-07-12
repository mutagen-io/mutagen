package remote

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/encoding"
	"github.com/havoc-io/mutagen/pkg/rsync"
)

const (
	// rsyncTransmissionGroupSize is the number of rsync transmissions that will
	// be grouped together and sent at once.
	rsyncTransmissionGroupSize = 10
)

// protobufRsyncEncoder adapts a Protocol Buffers encoder to the rsync.Encoder
// interface.
type protobufRsyncEncoder struct {
	// encoder is the underlying Protocol Buffers encoder.
	encoder *encoding.ProtobufEncoder
	// buffered is the number of transmissions currently buffered.
	buffered int
	// error indicates any previous encountered transmission error. It renders
	// the encoder inoperable.
	error error
}

func newProtobufRsyncEncoder(encoder *encoding.ProtobufEncoder) *protobufRsyncEncoder {
	return &protobufRsyncEncoder{encoder: encoder}
}

// Encode implements transmission encoding.
func (e *protobufRsyncEncoder) Encode(transmission *rsync.Transmission) error {
	// Check for previous errors.
	if e.error != nil {
		return errors.Wrap(e.error, "previous error encountered")
	}

	// Encode the transmission without sending.
	if err := e.encoder.EncodeWithoutFlush(transmission); err != nil {
		e.error = errors.Wrap(err, "unable to encode transmission")
		return e.error
	}

	// Increment the buffered message count.
	e.buffered++

	// If we've reached the maximum number of transmissions that we're willing
	// to buffer, then flush the buffer and reset the count.
	if e.buffered == rsyncTransmissionGroupSize {
		if err := e.encoder.Flush(); err != nil {
			e.error = errors.Wrap(err, "unable to write encoded messages")
			return e.error
		}
		e.buffered = 0
	}

	// Success.
	return nil
}

// Finalize flushes any buffered transmissions.
func (e *protobufRsyncEncoder) Finalize() error {
	// If there was previously an encoding error, then it will have propagated
	// up the chain and canceled rsync transmission, but we definitely shouldn't
	// attempt a flush if we know the stream is bad.
	if e.error != nil {
		return errors.Wrap(e.error, "previous error encountered")
	}

	// Flush any pending messages.
	if err := e.encoder.Flush(); err != nil {
		return errors.Wrap(err, "unable to write encoded messages")
	}

	// Reset the buffered message count, in case this encoder is reused.
	e.buffered = 0

	// Success.
	return nil
}

// protobufRsyncDecoder adapts a Protocol Buffers decoder to the rsync.Decoder
// interface.
type protobufRsyncDecoder struct {
	// decoder is the underlying Protocol Buffers decoder.
	decoder *encoding.ProtobufDecoder
}

func newProtobufRsyncDecoder(decoder *encoding.ProtobufDecoder) *protobufRsyncDecoder {
	return &protobufRsyncDecoder{decoder: decoder}
}

// Decode implements transmission decoding.
func (d *protobufRsyncDecoder) Decode(transmission *rsync.Transmission) error {
	// TODO: This is not particularly efficient because the Protocol Buffers
	// decoding implementation doesn't reuse existing capacity in operation data
	// buffers. This is something that needs to be fixed upstream, but we should
	// file an issue. Once it's done, nothing on our end needs to change except
	// to update the Protocol Buffers runtime.
	return d.decoder.Decode(transmission)
}

// Finalize is a no-op for Protocol Buffers rsync decoders.
func (d *protobufRsyncDecoder) Finalize() error {
	return nil
}
