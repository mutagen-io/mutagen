package rsync

import (
	"github.com/pkg/errors"
)

// EnsureValid ensures that the Transmission's invariants are respected.
func (t *Transmission) EnsureValid() error {
	// A nil transmission is not valid.
	if t == nil {
		return errors.New("nil transmission")
	}

	// Handle validation based on whether or not the operation is marked as
	// done.
	if t.Done {
		if t.Operation != nil {
			return errors.New("operation present at end of stream")
		}
	} else {
		if t.Operation == nil {
			return errors.New("operation missing from middle of stream")
		} else if err := t.Operation.EnsureValid(); err != nil {
			return errors.New("invalid operation in stream")
		} else if t.Error != "" {
			return errors.New("error in middle of stream")
		}
	}

	// Success.
	return nil
}
