package remote

import (
	"encoding/gob"

	"github.com/havoc-io/mutagen/pkg/rsync"
)

// gobRsyncEncoder adapts a gob encoder to the rsync.Encoder interface.
type gobRsyncEncoder struct {
	// encoder is the underlying gob encoder.
	encoder *gob.Encoder
}

// Encode implements transmission encoding.
func (e *gobRsyncEncoder) Encode(transmission *rsync.Transmission) error {
	return e.encoder.Encode(transmission)
}

// Finalize is a no-op for gob rsync encoders.
func (e *gobRsyncEncoder) Finalize() {}

// gobRsyncDecoder adapts a gob decoder to the rsync.Decoder interface.
type gobRsyncDecoder struct {
	decoder *gob.Decoder
}

// Decode implements transmission decoding.
func (d *gobRsyncDecoder) Decode(transmission *rsync.Transmission) error {
	return d.decoder.Decode(transmission)
}

// Finalize is a no-op for gob rsync decoders.
func (d *gobRsyncDecoder) Finalize() {}
