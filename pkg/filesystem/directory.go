package filesystem

import (
	"os"

	"github.com/pkg/errors"
)

func DirectoryContents(path string) ([]string, error) {
	// Open the directory and ensure its closure.
	directory, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "unable to open directory")
	}
	defer directory.Close()

	// Grab the directory names.
	names, err := directory.Readdirnames(0)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read directory names")
	}

	// Success.
	return names, nil
}
