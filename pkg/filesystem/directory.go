package filesystem

import (
	"fmt"
	"os"
)

// DirectoryContentsByPath returns the contents of the directory at the
// specified path. The ordering of the contents is non-deterministic.
func DirectoryContentsByPath(path string) ([]os.FileInfo, error) {
	// Open the directory and ensure its closure.
	directory, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open directory: %w", err)
	}
	defer directory.Close()

	// Grab the directory contents.
	contents, err := directory.Readdir(0)
	if err != nil {
		return nil, fmt.Errorf("unable to read directory contents: %w", err)
	}

	// Success.
	return contents, nil
}
