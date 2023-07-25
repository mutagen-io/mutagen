package ignore

import (
	"errors"
	"unicode"
)

// EnsurePatternValid ensures that the provided pattern is valid under general
// ignore syntax rules. A specific syntax may enforce additional validation, but
// should always use this function to perform a baseline validation. This
// function can also serve to perform validation in cases where the ignore
// syntax is not known or available.
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
	} else if pattern == "//" || pattern == "!//" {
		return errors.New("root directory pattern")
	}

	// Ensure that the pattern is not entirely space characters.
	var haveNonSpace bool
	for _, r := range pattern {
		if !unicode.IsSpace(r) {
			haveNonSpace = true
			break
		}
	}
	if !haveNonSpace {
		return errors.New("pattern is entirely space characters")
	}

	// Success.
	return nil
}

// IgnoreStatus encodes the different potential ignoredness states of content
// during filesystem traversal.
type IgnoreStatus uint8

const (
	// IgnoreStatusNominal indicates that content is neither explicitly ignored
	// nor unignored by any rule in an ignorer. In this case, the content is
	// typically traversed and processed, except in cases where an ignore
	// previously applied to parent content.
	IgnoreStatusNominal IgnoreStatus = iota
	// IgnoreStatusIgnored indicates that content is explicitly ignored.
	IgnoreStatusIgnored
	// IgnoreStatusUnignored indicates that content is explicitly unignored.
	IgnoreStatusUnignored
)

// Ignorer performs ignore evaluation during filesystem traversal. It takes the
// path being traversed and an indication of whether or not traversal should
// continue to child entries due to the possibility of content being unignored
// at a lower depth.
type Ignorer interface {
	// Ignore determines whether or not a filesystem entry should be ignored
	// based on its path and nature as a directory. It returns the ignore status
	// for the entry, as well as whether or not any inverted ignores indicate
	// that content beneath the entry could be explicitly unignored (and thus
	// that traversal should continue across this entry if it's a directory).
	// The path provided to Ignore will be relative to the synchronization root
	// and suitable for use with the fastpath package. The Ignore method should
	// always return the same results for a given set of arguments. Traversal
	// continuation should only be suggested if the entry is a directory and
	// should be suggested correctly regardless of ignore status, including in
	// the case of ignoreStatusNominal, where an ignore mask on the traversal
	// stack could cause the directory to be ignored.
	Ignore(path string, directory bool) (IgnoreStatus, bool)
}

// IgnoreCacheKey represents a key in an ignore cache.
type IgnoreCacheKey struct {
	// Path is the path used for testing ignore status.
	Path string
	// Directory is whether or not that path was a directory.
	Directory bool
}

// IgnoreCacheValue represents a value in an ignore cache.
type IgnoreCacheValue struct {
	// Status is the ignore status.
	Status IgnoreStatus
	// ContinueTraversal indicates whether or not to continue traversal in the
	// event that the content is ignored (due to either an explicit or implicit
	// (ignore-masked) ignore).
	ContinueTraversal bool
}

// IgnoreCache provides an efficient mechanism to avoid recomputing ignores.
type IgnoreCache map[IgnoreCacheKey]IgnoreCacheValue
