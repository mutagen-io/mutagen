package mutagen

import (
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
)

const (
	// BuildDirectoryName is the name of the build directory to create inside
	// the root of the Mutagen source tree.
	BuildDirectoryName = "build"
)

// SourceTreePath computes the path to the Mutagen source directory.
func SourceTreePath() (string, error) {
	// Compute the path to this script.
	_, filePath, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("unable to compute script path")
	}

	// Compute the path to the Mutagen source directory.
	return filepath.Dir(filepath.Dir(filepath.Dir(filePath))), nil
}
