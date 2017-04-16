package rsync

import (
	"github.com/pkg/errors"
)

// request encodes a batch request.
type request struct {
	// Paths are the requested paths.
	Paths []string
	// Signatures are the corresponding base signatures.
	Signatures []Signature
}

func (r request) ensureValid() error {
	// Ensure a 1-1 match between requested paths and signatures.
	if len(r.Paths) != len(r.Signatures) {
		return errors.New("path count does not match signature count")
	}

	// Success.
	return nil
}

// response encodes a stream response. It is used in the internal client/server
// protocol.
type response struct {
	// Done indicates that the response stream for this request is finished. If
	// set, there will be no operation in the response, but there may be an
	// error.
	Done bool
	// Operation is the next operation in the stream.
	Operation Operation
	// Error indicates that a non-terminal error has occurred. It will only be
	// present if Done is true.
	Error string
}
