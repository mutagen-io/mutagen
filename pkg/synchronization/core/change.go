package core

import (
	"errors"
	"fmt"
)

// EnsureValid ensures that Change's invariants are respected. If synchronizable
// is true, then unsynchronizable content in either the Old or New field will be
// considered invalid.
func (c *Change) EnsureValid(synchronizable bool) error {
	// A nil change is not valid.
	if c == nil {
		return errors.New("nil change")
	}

	// Technically we could validate the path, but that's error-prone,
	// expensive, and not really needed for memory safety. We also can't enforce
	// that the old entry value is not equal to the new entry value because for
	// the "synthetic" changes generated in unidirectional synchronization they
	// may be identical.

	// Validate entries.
	if err := c.Old.EnsureValid(synchronizable); err != nil {
		return fmt.Errorf("invalid old entry: %w", err)
	} else if err = c.New.EnsureValid(synchronizable); err != nil {
		return fmt.Errorf("invalid new entry: %w", err)
	}

	// Success.
	return nil
}

// slim creates a "slim" copy of the Change object, where both entries are slim
// copies with contents excluded.
func (c *Change) slim() *Change {
	return &Change{
		Path: c.Path,
		Old:  c.Old.Copy(EntryCopyBehaviorSlim),
		New:  c.New.Copy(EntryCopyBehaviorSlim),
	}
}

// IsRootDeletion indicates whether or not the change represents a root
// deletion.
func (c *Change) IsRootDeletion() bool {
	return c.Path == "" && c.Old != nil && c.New == nil
}

// IsRootTypeChange indicates whether or not the change represents a root type
// change.
func (c *Change) IsRootTypeChange() bool {
	return c.Path == "" && c.Old != nil && c.New != nil && c.Old.Kind != c.New.Kind
}
