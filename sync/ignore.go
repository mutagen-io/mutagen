package sync

import (
	"github.com/pkg/errors"

	"github.com/bmatcuk/doublestar"
)

type ignorePattern struct {
	negated bool
	pattern string
}

func newIgnorePattern(pattern string) (*ignorePattern, error) {
	// If the pattern is empty, it's invalid.
	if pattern == "" {
		return nil, errors.New("empty pattern")
	}

	// Check if this is a negated pattern. If so, strip off but record the
	// negation. Since we assume UTF-8, we can assume the '!' character will be
	// a single-byte rune.
	negated := false
	if pattern[0] == '!' {
		negated = true
		pattern = pattern[1:]
	}

	// Attempt to do a match with the pattern to ensure validity. We have to
	// match against a non-empty path (we choose something simple), otherwise
	// bad pattern errors won't be detected.
	if _, err := doublestar.Match(pattern, "a"); err != nil {
		return nil, errors.Wrap(err, "unable to validate pattern")
	}

	// Success.
	return &ignorePattern{negated, pattern}, nil
}

func (i *ignorePattern) matches(path string) bool {
	// Check if there is a match. Since we've already validated the pattern in
	// the constructor, we know match can't fail with an error (it's only return
	// code is on bad patterns).
	match, _ := doublestar.Match(i.pattern, path)
	return match
}

func ValidIgnorePattern(pattern string) bool {
	// Verify that we can parse the ignore.
	_, err := newIgnorePattern(pattern)
	return err == nil
}

type ignorer struct {
	patterns []*ignorePattern
}

func newIgnorer(patterns []string) (*ignorer, error) {
	// Parse patterns.
	ignorePatterns := make([]*ignorePattern, len(patterns))
	for i, p := range patterns {
		if ip, err := newIgnorePattern(p); err != nil {
			return nil, errors.Wrap(err, "unable to parse pattern")
		} else {
			ignorePatterns[i] = ip
		}
	}

	// Success.
	return &ignorer{ignorePatterns}, nil
}

func (i *ignorer) ignored(path string) bool {
	// Nothing is initially ignored.
	ignored := false

	// Run through patterns, keeping track of the ignored state as we reach more
	// specific rules.
	for _, p := range i.patterns {
		// If there's no match, then this rule doesn't apply.
		if !p.matches(path) {
			continue
		}

		// If we have a match, then change the ignored state based on whether or
		// not the pattern is negated.
		ignored = !p.negated
	}

	// Done.
	return ignored
}
