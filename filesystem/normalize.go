package filesystem

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/pkg/errors"
)

func tildeExpand(path string) (string, error) {
	// Only process relevant paths.
	if len(path) < 3 || path[0] != '~' {
		return path, nil
	}

	// If the second character isn't a path separator, then someone is probably
	// trying to do a ~username expansion, but we can't support that without CGO
	// to support user.Lookup.
	if !os.IsPathSeparator(path[1]) {
		return "", errors.New("unable to perform user lookup")
	}

	// Grab the current user.
	self, err := user.Current()
	if err != nil {
		return "", errors.Wrap(err, "unable to access user information")
	}

	// Compute the path.
	return filepath.Join(self.HomeDir, path[2:]), nil
}

func Normalize(path string) (string, error) {
	// Expand any leading tilde.
	path, err := tildeExpand(path)
	if err != nil {
		return "", errors.Wrap(err, "unable to perform tilde expansion")
	}

	// Convert to an absolute path.
	path, err = filepath.Abs(path)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute absolute path")
	}

	// Evaluate any symlinks.
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", errors.Wrap(err, "unable to evaluate symlinks")
	}

	// Success.
	return path, nil
}
