package docker

import (
	"errors"
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore/docker/internal/third_party/patternmatcher"
)

// EnsurePatternValid ensures that the provided pattern is valid under
// Docker-style ignore syntax.
func EnsurePatternValid(pattern string) error {
	// Check for invalid patterns, specifically those that would leave us with
	// an empty string after parsing or those that would exclude the entire
	// synchronization root. Obviously we can't perform complete validation for
	// all patterns, but if they pass this parsing, they should be sane enough
	// to at least try to parse and match.
	if pattern == "" || pattern == "!" {
		return errors.New("empty pattern")
	} else if pattern == "/" || pattern == "!/" {
		return errors.New("root pattern")
	}

	// Verify that the pattern meets the requirements of the patternmatcher
	// package. There's no patternmatcher.Pattern constructor, and its internal
	// compilation operation is lazy, so we have to construct the full matcher
	// and then attempt to perform a match.
	if pm, err := patternmatcher.New([]string{pattern}); err != nil {
		return err
	} else if _, err := pm.MatchesOrParentMatches("path"); err != nil {
		return err
	}

	// Success.
	return nil
}

// ignorer implements ignore.Ignorer for Docker-style ignores.
type ignorer struct {
	// matcher is the underlying pattern matcher.
	matcher *patternmatcher.PatternMatcher
}

// NewIgnorer creates a new ignorer using Docker-style ignore patterns.
func NewIgnorer(patterns []string) (ignore.Ignorer, error) {
	// Create the pattern matcher.
	matcher, err := patternmatcher.New(patterns)
	if err != nil {
		return nil, fmt.Errorf("unable to construct pattern matcher: %w", err)
	}

	// Done.
	return &ignorer{matcher}, nil
}

// Ignore implements ignore.Ignorer.ignore.
func (i *ignorer) Ignore(path string, directory bool) (ignore.IgnoreStatus, bool) {
	// Pass the matching operation to the underlying matcher.
	status, continueTraversal := i.matcher.MatchesForMutagen(path, directory)

	// Adapt the potential match statuses.
	switch status {
	case patternmatcher.MatchStatusNominal:
		return ignore.IgnoreStatusNominal, continueTraversal
	case patternmatcher.MatchStatusMatched:
		return ignore.IgnoreStatusIgnored, continueTraversal
	case patternmatcher.MatchStatusInverted:
		return ignore.IgnoreStatusUnignored, continueTraversal
	default:
		panic("unhandled patternmatcher status")
	}
}
