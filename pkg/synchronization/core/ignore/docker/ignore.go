package docker

import (
	"errors"
	"fmt"
	pathpkg "path"
	"strings"

	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore/docker/internal/third_party/patternmatcher"
)

// newValidatedPatternMatcher constructs a Mutagen-validated PatternMatcher
// instance using the specified pattern. This wrapper helps to consolidate
// validation for EnsurePatternValid and NewIgnorer, which is complicated due to
// the fact that patternmatcher.Pattern isn't designed to be used individually.
func newValidatedPatternMatcher(patterns []string) (*patternmatcher.PatternMatcher, error) {
	// Perform initial validation and cleaning. The modifications we make here
	// are primarily to keep alignment with .dockerignore parsing, which does
	// some preprocessing to remove whitespace and slash prefixes:
	// https://github.com/moby/buildkit/blob/18fc875d9bfd6e065cd8211abc639434ba65aa56/frontend/dockerfile/dockerignore/dockerignore.go#L38-L57
	// Additional cleaning and processing is performed when constructing a
	// PatternMatcher, but observable .dockerignore behavior relies on both.
	cleanedPatterns := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		// Disable escaping since we need our patterns to be portable. This also
		// means that paths containing backslash separators, which are normally
		// tolerated (though rarely used) in a .dockerignore file on Windows,
		// will be rejected here in favor of portability.
		if strings.IndexByte(pattern, '\\') >= 0 {
			return nil, errors.New("escape sequences and backslash-separated paths disallowed in portable .dockerignore patterns")
		}

		// Trim whitespace from the pattern and check for empty patterns. Docker
		// will just ignore these, but since they have no meaning, we'll raise
		// an error to avoid misconfiguration.
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			return nil, errors.New("whitespace-only pattern")
		}

		// If the pattern is negated, then strip off the negation before doing
		// any additional cleaning.
		negated := pattern[0] == '!'
		if negated {
			pattern = strings.TrimSpace(pattern[1:])
			if pattern == "" {
				return nil, errors.New("whitespace-only negated pattern")
			}
		}

		// Perform a cleaning operation on the path and watch out for root
		// paths. We already assume that we're working with forward slashes
		// given that we exclude backslashes above, so we can just perform a
		// standard path.Clean.
		//
		// We could potentially allow "!/" patterns (i.e. allow a root path
		// specification if this is a negated pattern), but there's no reason to
		// do that because we don't allow the root to be excluded. Thus, it's
		// best to flag this odd specification. It also saves us the complexity
		// the empty string edge case (after the slash is stripped off). In any
		// case, such a pattern would never end up matching anything because the
		// root path is never evaluated for ignoring in Scan.
		pattern = pathpkg.Clean(pattern)
		if pattern == "/" {
			return nil, errors.New("root pattern")
		}

		// Remove any prefix slash on the pattern. All Docker-style matching is
		// inherently root-relative, but it won't match if a slash prefix is
		// present, instead relying on that slash having been removed when the
		// .dockerignore file was loaded.
		if len(pattern) > 1 && pattern[0] == '/' {
			pattern = pattern[1:]
		}

		// Replace the negation, if necessary.
		if negated {
			pattern = "!" + pattern
		}

		// Record the cleaned pattern.
		cleanedPatterns = append(cleanedPatterns, pattern)
	}

	// Create the resulting matcher and enforce pre-compilation.
	matcher, err := patternmatcher.New(cleanedPatterns)
	if err != nil {
		return nil, err
	} else if err = matcher.PrecompileForMutagen(); err != nil {
		return nil, err
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
