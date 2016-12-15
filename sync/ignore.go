package sync

import (
	"github.com/pkg/errors"

	"github.com/bmatcuk/doublestar"
)

type ignorer struct {
	negated bool
	pattern string
}

func newIgnorer(pattern string) (ignorer, error) {
	// If the pattern is empty, it's invalid.
	if pattern == "" {
		return ignorer{}, errors.New("empty pattern")
	}

	// Check if this is a negated pattern. If so, strip off the negation.
	negated := false
	if pattern[0] == '!' {
		negated = true
		pattern = pattern[1:]
	}

	// Attempt to do a match with the pattern to ensure validity.
	if _, err := doublestar.Match(pattern, ""); err != nil {
		return ignorer{}, errors.Wrap(err, "unable to validate pattern")
	}

	// Success.
	return ignorer{negated, pattern}, nil
}

func (i ignorer) matches(path string) bool {
	// Check if there is a match. Since we've already validated the pattern in
	// the constructor, we know match can't fail with an error (it's only return
	// code is on bad patterns).
	match, _ := doublestar.Match(i.pattern, path)
	return match
}

func ValidIgnore(pattern string) bool {
	// Verify that we can parse the ignore.
	_, err := newIgnorer(pattern)
	return err == nil
}
