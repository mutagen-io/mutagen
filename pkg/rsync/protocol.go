package rsync

import (
	"github.com/pkg/errors"
)

// Transmission represents a single message in a transmission stream. Its
// internals are public to allow for transmission using a reflection-based
// encoder (such as gob), but it should otherwise be treated as an opaque type
// with a private implementation.
type Transmission struct {
	// Done indicates that the operation stream for the current file is
	// finished. If set, there will be no operation in the response, but there
	// may be an error.
	Done bool
	// Operation is the next operation in the stream for the current file.
	Operation Operation
	// Error indicates that a non-terminal error has occurred. It will only be
	// present if Done is true.
	Error string
}

// ensureValid ensures that the Transmission's invariants are respected.
func (t Transmission) ensureValid() error {
	// Handle validation based on whether or not the operation is marked as
	// done.
	if t.Done {
		if !t.Operation.isZeroValue() {
			return errors.New("non-zero operation at end of stream")
		}
	} else {
		if t.Operation.isZeroValue() {
			return errors.New("zero-value operation in middle of stream")
		} else if t.Error != "" {
			return errors.New("non-empty error in middle of stream")
		}
	}

	// Success.
	return nil
}
