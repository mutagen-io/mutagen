package filesystem

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/pkg/errors"
)

// tildeExpand attempts tilde expansion of paths beginning with ~/ or
// ~<username>/. On Windows, it additionally supports ~\ and ~<username>\.
func tildeExpand(path string) (string, error) {
	// Only process relevant paths.
	if path == "" || path[0] != '~' {
		return path, nil
	}

	// Find the first character in the path that's considered a path separator
	// on the platform. Path seperators are always single-byte, so we can safely
	// loop over the path's bytes.
	pathSeparatorIndex := -1
	for i := 0; i < len(path); i++ {
		if os.IsPathSeparator(path[i]) {
			pathSeparatorIndex = i
			break
		}
	}

	// Divide the path into the "username" portion and the "subpath" portion -
	// i.e. those portions coming before and after the separator, respectively.
	var username string
	var remaining string
	if pathSeparatorIndex > 0 {
		username = path[1:pathSeparatorIndex]
		remaining = path[pathSeparatorIndex+1:]
	} else {
		username = path[1:]
	}

	// Compute the relevant home directory. If the username is empty, then we
	// use the current user's home directory, otherwise we need to do a lookup.
	var homeDirectory string
	if username == "" {
		if h, err := os.UserHomeDir(); err != nil {
			return "", errors.Wrap(err, "unable to compute path to home directory")
		} else {
			homeDirectory = h
		}
	} else {
		if u, err := user.Lookup(username); err != nil {
			return "", errors.Wrap(err, "unable to lookup user")
		} else {
			homeDirectory = u.HomeDir
		}
	}

	// Compute the full path.
	return filepath.Join(homeDirectory, remaining), nil
}

// Normalize normalizes a path, expanding home directory tildes, converting it
// to an absolute path, and cleaning the result.
func Normalize(path string) (string, error) {
	// Expand any leading tilde.
	path, err := tildeExpand(path)
	if err != nil {
		return "", errors.Wrap(err, "unable to perform tilde expansion")
	}

	// Convert to an absolute path. This will also invoke filepath.Clean.
	path, err = filepath.Abs(path)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute absolute path")
	}

	// Success.
	return path, nil
}
