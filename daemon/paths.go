package daemon

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/filesystem"
)

const (
	daemonDirectoryName = "daemon"
)

func subpath(name string) (string, error) {
	// Compute the daemon root directory path and ensure it exists.
	daemonRoot, err := filesystem.Mutagen(true, daemonDirectoryName)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute daemon directory")
	}

	// Compute the combined path.
	return filepath.Join(daemonRoot, name), nil
}
