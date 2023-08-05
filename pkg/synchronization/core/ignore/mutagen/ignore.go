package mutagen

import (
	"errors"
	"fmt"
	pathpkg "path"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore"
)

// cleanPreservingTrailingSlash is a variant of path.Clean that preserves
// trailing slashes.
func cleanPreservingTrailingSlash(path string) string {
	// Determine whether or not a trailing slash exists. We enforce a minimum
	// length to ensure that we're not dealing with "/".
	var needTrailingSlash bool
	if l := len(path); l > 1 {
		needTrailingSlash = path[l-1] == '/'
	}

	// Perform a clean operation, adjusting the result as necessary.
	if result := pathpkg.Clean(path); needTrailingSlash {
		return result + "/"
	} else {
		return result
	}
}

// EnsurePatternValid ensures that the provided pattern is valid under
// Mutagen-style ignore syntax.
func EnsurePatternValid(pattern string) error {
	_, err := newIgnorePattern(pattern)
	return err
}

// ignorePattern represents a single parsed Mutagen-style ignore pattern.
type ignorePattern struct {
	// negated indicates whether or not the pattern is negated.
	negated bool
	// directoryOnly indicates whether or not the pattern should only match
	// directories.
	directoryOnly bool
	// matchLeaf indicates whether or not the pattern should be matched against
	// a path's base name in addition to the whole path.
	matchLeaf bool
	// pattern is the pattern to use in matching.
	pattern string
}

// newIgnorePattern validates and parses a user-provided ignore pattern.
func newIgnorePattern(pattern string) (*ignorePattern, error) {
	// Ensure that the pattern is not empty.
	if len(pattern) == 0 {
		return nil, errors.New("empty pattern")
	}

	// Check if this is a negated pattern. If so, remove the exclamation point
	// prefix, since it won't enter into pattern matching. Take this opportunity
	// to ensure that we didn't receive an empty negated pattern.
	var negated bool
	if pattern[0] == '!' {
		negated = true
		pattern = pattern[1:]
	}
	if pattern == "" {
		return nil, errors.New("negated empty pattern")
	}

	// Perform a cleaning operation on the pattern, making sure to preserve any
	// trailing slashes.
	pattern = cleanPreservingTrailingSlash(pattern)

	// Ensure that we haven't received a pattern targeting the synchronization
	// root.
	//
	// We could potentially allow "!/" or "!//" patterns (i.e. allow a root path
	// or root directory path specification if this is a negated pattern), but
	// there's no reason to do that because we don't allow the root to be
	// excluded. Thus, it's best to flag this odd specification. It also saves
	// us the complexity the empty string edge case (after the slash/slashes
	// is/are stripped off). In any case, such a pattern would never end up
	// matching anything because the root path is never evaluated for ignoring
	// in Scan.
	if pattern == "/" {
		return nil, errors.New("root pattern")
	} else if pattern == "//" {
		return nil, errors.New("root directory pattern")
	}

	// Check if this is an absolute pattern. If so, remove the forward slash
	// prefix, since it won't enter into pattern matching.
	var absolute bool
	if pattern[0] == '/' {
		absolute = true
		pattern = pattern[1:]
	}

	// Check if this is a directory-only pattern. If so, remove the trailing
	// slash, since it won't enter into pattern matching.
	var directoryOnly bool
	if pattern[len(pattern)-1] == '/' {
		directoryOnly = true
		pattern = pattern[:len(pattern)-1]
	}

	// Determine whether or not the pattern contains a slash.
	containsSlash := strings.IndexByte(pattern, '/') >= 0

	// Attempt to do a match with the pattern to ensure validity. We have to
	// match against a non-empty path (we choose something simple), otherwise
	// bad pattern errors won't be detected.
	if _, err := doublestar.Match(pattern, "a"); err != nil {
		return nil, fmt.Errorf("unable to validate pattern: %w", err)
	}

	// Success.
	return &ignorePattern{
		negated:       negated,
		directoryOnly: directoryOnly,
		matchLeaf:     (!absolute && !containsSlash),
		pattern:       pattern,
	}, nil
}

// matches indicates whether or not the ignore pattern matches the specified
// path and metadata.
func (i *ignorePattern) matches(path string, directory bool) bool {
	// If this pattern only applies to directories and this is not a directory,
	// then this is not a match.
	if i.directoryOnly && !directory {
		return false
	}

	// Check if there is a direct match. Since we've already validated the
	// pattern in the constructor, we know match can't fail with an error (it's
	// only return code is on bad patterns).
	if match, _ := doublestar.Match(i.pattern, path); match {
		return true
	}

	// If it makes sense, attempt to match on the last component of the path,
	// assuming the path is non-empty (non-root).
	if i.matchLeaf && path != "" {
		if match, _ := doublestar.Match(i.pattern, pathpkg.Base(path)); match {
			return true
		}
	}

	// No match.
	return false
}

// ignorer implements ignore.Ignorer for Mutagen-style ignores.
type ignorer struct {
	// patterns are the underlying ignore patterns.
	patterns []*ignorePattern
	// negatedPatternCount is the number of patterns in the ignorer that are
	// negated patterns.
	negatedPatternCount uint
}

// NewIgnorer creates a new ignorer using Mutagen-style ignore patterns.
func NewIgnorer(patterns []string) (ignore.Ignorer, error) {
	// Parse patterns.
	ignorePatterns := make([]*ignorePattern, len(patterns))
	var negatedPatternCount uint
	for i, pattern := range patterns {
		if p, err := newIgnorePattern(pattern); err != nil {
			return nil, fmt.Errorf("unable to parse pattern: %w", err)
		} else {
			ignorePatterns[i] = p
			if p.negated {
				negatedPatternCount++
			}
		}
	}

	// Success.
	return &ignorer{
		patterns:            ignorePatterns,
		negatedPatternCount: negatedPatternCount,
	}, nil
}

// Ignore implements ignore.Ignorer.ignore.
func (i *ignorer) Ignore(path string, directory bool) (ignore.IgnoreStatus, bool) {
	// Start with a nominal ignore status.
	var status ignore.IgnoreStatus

	// Run through the ignore patterns, updating the ignore state as we reach
	// more specific rules.
	negatedPatternsRemaining := i.negatedPatternCount
	for _, pattern := range i.patterns {
		// See if we can skip the (relatively expensive) matching process. If
		// we're already in an ignored state and there aren't any negated
		// patterns remaining, then we can't leave that state, and thus we can
		// skip any further matching. If this pattern is negated, then we'll
		// decrement the remaining negated pattern count, and we can also skip
		// matching for this particular pattern if we're already in an unignored
		// state. Finally, if we're already in an ignored state and this is a
		// non-negated pattern, then we also won't change state as a result of
		// this particular pattern and can skip matching.
		if status == ignore.IgnoreStatusIgnored && negatedPatternsRemaining == 0 {
			break
		} else if pattern.negated {
			negatedPatternsRemaining--
			if status == ignore.IgnoreStatusUnignored {
				continue
			}
		} else if status == ignore.IgnoreStatusIgnored {
			continue
		}

		// Perform a matching operation and adjust the status as appropriate.
		if !pattern.matches(path, directory) {
			continue
		} else if pattern.negated {
			status = ignore.IgnoreStatusUnignored
		} else {
			status = ignore.IgnoreStatusIgnored
		}
	}

	// For Mutagen-style ignores, we always continue traversal in the case of
	// nominal or unignored content.
	if directory && (status == ignore.IgnoreStatusNominal || status == ignore.IgnoreStatusUnignored) {
		return status, true
	}

	// For non-directory types, or ignored directories, traversal continuation
	// is always false.
	return status, false
}
