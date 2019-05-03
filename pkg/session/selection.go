package session

import (
	"github.com/pkg/errors"
)

// EnsureValid verifies that a Selection is valid.
func (s *Selection) EnsureValid() error {
	// A nil selection is not valid.
	if s == nil {
		return errors.New("nil selection")
	}

	// Count the number of session selection mechanisms present.
	var mechanismsPresent uint
	if s.All {
		mechanismsPresent++
	}
	if len(s.Specifications) > 0 {
		mechanismsPresent++
	}
	if s.LabelSelector != "" {
		mechanismsPresent++
	}

	// Enforce that exactly one session selection mechanism is present.
	if mechanismsPresent > 1 {
		return errors.New("multiple selection mechanisms present")
	} else if mechanismsPresent < 1 {
		return errors.New("no selection mechanisms present")
	}

	// We can't validate specifications, if present, because they don't conform
	// to any restricted patterns, but they don't pose a risk for our matching
	// algorithm.

	// We explicitly avoiding validating the label selector, if present, because
	// it doesn't pose a risk to parse unvalidated and it would only be possible
	// to validate by parsing, so we'll catch any format errors later.

	// Success.
	return nil
}
