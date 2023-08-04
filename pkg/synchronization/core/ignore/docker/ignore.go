package docker

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore/docker/internal/third_party/patternmatcher"
)

// newValidatedPatternMatcher constructs a Mutagen-validated PatternMatcher
// instance using the specified pattern. This wrapper helps to consolidate
// validation for EnsurePatternValid and NewIgnorer, which is complicated due to
// the fact that patternmatcher.Pattern isn't designed to be used individually.
func newValidatedPatternMatcher(patterns []string) (*patternmatcher.PatternMatcher, error) {
	// Disable escaping since we need our patterns to be portable.
	for _, pattern := range patterns {
		if strings.IndexByte(pattern, '\\') >= 0 {
			return nil, errors.New("escape sequences disallowed in portable .dockerignore patterns")
		}
	}

	// Create the resulting matcher and enforce pre-compilation.
	matcher, err := patternmatcher.New(patterns)
	if err != nil {
		return nil, err
	} else if err = matcher.PrecompileForMutagen(); err != nil {
		return nil, err
	}

	// Enforce that none of the resulting patterns target a root path. Note that
	// the patternmatcher library will convert to platform-native slashes
	// internally, so we have to check for standalone separator accordingly.
	for _, pattern := range matcher.Patterns() {
		if pattern.String() == string(filepath.Separator) {
			return nil, errors.New("root pattern")
		}
	}

	// Success.
	return matcher, nil
}

// EnsurePatternValid ensures that the provided pattern is valid under
// Docker-style ignore syntax.
func EnsurePatternValid(pattern string) error {
	_, err := newValidatedPatternMatcher([]string{pattern})
	return err
}

// ignorer implements ignore.Ignorer for Docker-style ignores.
type ignorer struct {
	// matcher is the underlying pattern matcher.
	matcher *patternmatcher.PatternMatcher
}

// NewIgnorer creates a new ignorer using Docker-style ignore patterns.
func NewIgnorer(patterns []string) (ignore.Ignorer, error) {
	// Create the pattern matcher and validate patterns.
	matcher, err := newValidatedPatternMatcher(patterns)
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
