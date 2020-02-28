package filesystem

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// LibexecPath computes the expected libexec path assuming a Filesystem
// Hierarchy Standard layout with the current executable located in the bin
// directory. It will return an error if the executable does not exist within
// the "bin" directory of such a layout, but it does not verify that the libexec
// directory exists.
func LibexecPath() (string, error) {
	// Compute the path to the current executable.
	executablePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("unable to compute executable path: %w", err)
	}

	// If the executable path is a symbolic link, then perform resolution.
	// Unfortunately there's no way to do this in a completely race-free
	// fashion, but we're dealing with system prefixes here so it shouldn't be a
	// problem.
	if metadata, err := os.Lstat(executablePath); err != nil {
		return "", fmt.Errorf("unable to read executable metadata: %w", err)
	} else if metadata.Mode()&os.ModeSymlink != 0 {
		if target, err := os.Readlink(executablePath); err != nil {
			return "", fmt.Errorf("unable to read executable symbolic link target: %w", err)
		} else if resolved, err := filepath.Abs(filepath.Join(filepath.Dir(executablePath), target)); err != nil {
			return "", fmt.Errorf("unable to resolve executable symbolic link target: %w", err)
		} else {
			executablePath = resolved
		}
	}

	// Check that the executable resides within a bin directory.
	if filepath.Base(filepath.Dir(executablePath)) != "bin" {
		return "", errors.New("executable does not reside within bin directory")
	}

	// Compute the expected libexec path.
	return filepath.Clean(filepath.Join(executablePath, "..", "..", "libexec")), nil
}
