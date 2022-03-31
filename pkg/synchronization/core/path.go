package core

import (
	"strings"
)

// pathJoinable converts a base path to be joinable via string concatenation.
func pathJoinable(base string) string {
	// Handle the case of the synchronization root.
	if base == "" {
		return ""
	}

	// Prepare the path for concatenation.
	return base + "/"
}

// pathDir is a fast alternative to path.Dir designed specifically for
// root-relative synchronization paths. It avoids the unnecessary path cleaning
// overhead incurred by path.Dir. Note that, unlike path.Dir, this function
// isn't equivalent to returning the first return value from path.Split, because
// in that case the trailing slash remains in the directory path. The provided
// path must be non-empty, otherwise this function will panic.
func pathDir(path string) string {
	// Disallow synchronization root paths.
	if path == "" {
		panic("empty path")
	}

	// Identify the index of the last slash in the path.
	lastSlashIndex := strings.LastIndexByte(path, '/')

	// If there is no slash, then the parent is the synchronization root.
	if lastSlashIndex == -1 {
		return ""
	}

	// Verify that the parent path isn't empty. There aren't any scenarios where
	// this is allowed.
	if lastSlashIndex == 0 {
		panic("empty parent path")
	}

	// Trim off the slash and everything that follows.
	return path[:lastSlashIndex]
}

// PathBase is a fast alternative to path.Base designed specifically for
// root-relative synchronization paths. If the provided path is empty (i.e. the
// root path), this function returns an empty string. If the provided path
// contains no slashes, then it is returned directly. If the path ends with a
// slash, this function panics, because that represents an invalid root-relative
// path.
func PathBase(path string) string {
	// If this is the root path, then just return an empty string.
	if path == "" {
		return ""
	}

	// Identify the index of the last slash in the path.
	lastSlashIndex := strings.LastIndexByte(path, '/')

	// If there is no slash, then the path is a file directly under the
	// synchronization root.
	if lastSlashIndex == -1 {
		return path
	}

	// Verify that the base name isn't empty (i.e. that the string doesn't end
	// with a slash). We could do additional validation here (e.g. validating
	// the path segment before the slash), but it would be costly and somewhat
	// unnecessary. This check is sufficient to ensure that this function can
	// return a meaningful answer.
	if lastSlashIndex == len(path)-1 {
		panic("empty base name")
	}

	// Extract the base name.
	return path[lastSlashIndex+1:]
}

// pathLess performs a sort comparison between two root-relative synchronization
// paths. It returns true if first comes before second in DFS traversal.
func pathLess(first, second string) bool {
	// Handle trivial cases first.
	if first == second {
		return false
	} else if first == "" {
		return true
	} else if second == "" {
		return false
	}

	// Compare the path components. We work hard to avoid allocations here since
	// this is a comparison function for sorting algorithms.
	for {
		// Extract the front path component from the first path.
		firstFirstSlashIndex := strings.IndexByte(first, '/')
		var firstFrontComponent string
		if firstFirstSlashIndex == -1 {
			firstFrontComponent = first
		} else {
			firstFrontComponent = first[:firstFirstSlashIndex]
		}

		// Extract the front path component from the second path.
		secondFirstSlashIndex := strings.IndexByte(second, '/')
		var secondFrontComponent string
		if secondFirstSlashIndex == -1 {
			secondFrontComponent = second
		} else {
			secondFrontComponent = second[:secondFirstSlashIndex]
		}

		// Compare the front path components.
		if firstFrontComponent < secondFrontComponent {
			return true
		} else if secondFrontComponent < firstFrontComponent {
			return false
		}

		// The front path components are equal. If either path has no remaining
		// components, then the comparison is complete, otherwise we move ahead
		// to the next path components. Note that we don't have to consider the
		// case where firstFirstSlashIndex and secondFirstSlashIndex are both -1
		// (with front components also equal) because that would mean the
		// strings were entirely equal, which we handle above.
		if firstFirstSlashIndex == -1 {
			return true
		} else if secondFirstSlashIndex == -1 {
			return false
		} else {
			first = first[firstFirstSlashIndex+1:]
			second = second[secondFirstSlashIndex+1:]
		}
	}
}
