package process

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
)

// FindCommand searches for a command with the specified name within the
// specified list of directories. It's similar to os/exec.LookPath, except that
// it allows one to manually specify paths, and it uses a slightly simpler
// lookup mechanism.
func FindCommand(name string, paths []string) (string, error) {
	// Iterate through the directories.
	for _, path := range paths {
		// Compute the target name.
		target := filepath.Join(path, ExecutableName(name, runtime.GOOS))

		// Check if the target exists and has the correct type.
		// TODO: Should we do more extensive (and platform-specific) testing on
		// the resulting metadata? See, e.g., the implementation of
		// os/exec.LookPath.
		if metadata, err := os.Stat(target); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", errors.Wrap(err, "unable to query file metadata")
		} else if metadata.Mode()&os.ModeType != 0 {
			continue
		} else {
			return target, nil
		}
	}

	// Failure.
	return "", errors.New("unable to locate command")
}
