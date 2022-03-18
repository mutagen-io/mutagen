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

	// Success.
	return nil
}

// Equal performs an equivalence comparison between this snapshot and another.
func (s *Snapshot) Equal(other *Snapshot) bool {
	return s.Content.Equal(other.Content, true) &&
		s.PreservesExecutability == other.PreservesExecutability &&
		s.DecomposesUnicode == other.DecomposesUnicode
}
