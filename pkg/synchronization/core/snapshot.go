package core

import (
	"errors"
	"fmt"
)

// EnsureValid ensures that Snapshot's invariants are respected.
func (s *Snapshot) EnsureValid() error {
	// A nil snapshot is not valid.
	if s == nil {
		return errors.New("nil snapshot")
	}

	// Ensure that the snapshot content is valid. Snapshots may naturally
	// contain unsynchronizable content.
	if err := s.Content.EnsureValid(false); err != nil {
		return fmt.Errorf("invalid snapshot content: %w", err)
	}

	// All values of behavioral metadata fields are valid.

	// While there are some validations that we could perform on the statistical
	// fields of the snapshot, they don't exhaustively determine validity.
	// Moreover, some of these might be expensive, such as comparing the static
	// entry counts with those generated by the content entry. On the whole,
	// since the statistical fields are only used for informational purposes,
	// and since they are all valid from a memory-safety perspective, there's no
	// point in validating them.

	// Success.
	return nil
}

// Equal performs an equivalence comparison between this snapshot and another.
func (s *Snapshot) Equal(other *Snapshot) bool {
	return s.Content.Equal(other.Content, true) &&
		s.PreservesExecutability == other.PreservesExecutability &&
		s.DecomposesUnicode == other.DecomposesUnicode
}
