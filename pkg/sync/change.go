package sync

import (
	"github.com/pkg/errors"
)

// copySlim creates a "slim" copy of the Change object, where both entries are
// shallow copies with contents excluded.
func (c *Change) copySlim() *Change {
	return &Change{
		Path: c.Path,
		Old:  c.Old.copySlim(),
		New:  c.New.copySlim(),
	}
}

// EnsureValid ensures that Change's invariants are respected.
func (c *Change) EnsureValid() error {
	// A nil change is not valid.
	if c == nil {
		return errors.New("nil change")
	}

	// Technically we could validate the path, but that's error prone,
	// expensive, and not really needed for memory safety. We also can't enforce
	// that the old entry value is not equal to the new entry value because for
	// the "synthetic" changes generated in unidirectional synchronization they
	// may be identical.

	// Success.
	return nil
}
